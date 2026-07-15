package iru

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/retryx"
	"github.com/goreleaser/goreleaser/v2/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const (
	defaultInstallType        = "package"
	defaultInstallEnforcement = "install_once"
	tokenEnvKey               = "IRU_API_TOKEN"
)

// Pipe for iru custom apps.
type Pipe struct{}

func (Pipe) String() string        { return "iru custom apps" }
func (Pipe) ContinueOnError() bool { return true }
func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Iru) || ctx.Config.Iru.URL == ""
}

func (Pipe) Default(ctx *context.Context) error {
	iru := &ctx.Config.Iru
	iru.Name = cmp.Or(iru.Name, ctx.Config.ProjectName)
	iru.InstallType = cmp.Or(iru.InstallType, defaultInstallType)
	iru.InstallEnforcement = cmp.Or(iru.InstallEnforcement, defaultInstallEnforcement)
	return nil
}

func (p Pipe) Publish(ctx *context.Context) error {
	cfg := ctx.Config.Iru

	t := tmpl.New(ctx)
	disabled, err := t.Bool(cfg.Disable)
	if err != nil {
		return fmt.Errorf("could not evaluate iru.disable: %w", err)
	}
	if disabled {
		return pipe.Skip("iru.disable is set")
	}

	if err := t.ApplyAll(
		&cfg.URL,
		&cfg.APIToken,
		&cfg.LibraryItemID,
		&cfg.SelfServiceCategoryID,
	); err != nil {
		return fmt.Errorf("could not apply templates: %w", err)
	}
	cfg.URL = strings.TrimRight(cfg.URL, "/")
	if cfg.URL == "" {
		return errors.New("url templated to an empty string")
	}
	cfg.APIToken = cmp.Or(cfg.APIToken, ctx.Env[tokenEnvKey])
	if cfg.APIToken == "" {
		return fmt.Errorf("missing API token: set iru.api_token or the $%s environment variable", tokenEnvKey)
	}
	if err := validate(cfg); err != nil {
		return err
	}

	artifacts := ctx.Artifacts.Filter(artifact.And(
		artifact.ByTypes(
			artifact.UploadableArchive,
			artifact.UploadableBinary,
			artifact.UploadableFile,
		),
		artifact.ByIDs(cfg.IDs...),
	)).List()
	if len(artifacts) == 0 {
		return pipe.Skip("no artifacts found matching the given filters")
	}
	if cfg.LibraryItemID != "" && len(artifacts) > 1 {
		return fmt.Errorf(
			"library_item_id is set, but %d artifacts matched: use iru.ids to select a single artifact",
			len(artifacts),
		)
	}

	g := semerrgroup.New(ctx.Parallelism)
	for _, art := range artifacts {
		g.Go(func() error {
			return p.publishArtifact(ctx, cfg, art)
		})
	}
	return g.Wait()
}

// validate checks the conditionally required field combinations documented
// by the Iru API, so misconfigurations fail before anything is uploaded.
func validate(cfg config.Iru) error {
	if cfg.InstallType == "zip" && cfg.UnzipLocation == "" {
		return errors.New("install_type is zip, but unzip_location is not set")
	}
	if cfg.InstallEnforcement == "continuously_enforce" && cfg.AuditScript == "" {
		return errors.New("install_enforcement is continuously_enforce, but audit_script is not set")
	}
	if cfg.ShowInSelfService != nil && *cfg.ShowInSelfService && cfg.SelfServiceCategoryID == "" {
		return errors.New("show_in_self_service is enabled, but self_service_category_id is not set")
	}
	return nil
}

func (p Pipe) publishArtifact(ctx *context.Context, cfg config.Iru, art *artifact.Artifact) error {
	name, err := tmpl.New(ctx).WithArtifact(art).Apply(cfg.Name)
	if err != nil {
		return fmt.Errorf("could not apply templates to iru.name: %w", err)
	}

	upload, err := p.initUpload(ctx, cfg, art.Name)
	if err != nil {
		return fmt.Errorf("could not initialize upload: %w", err)
	}

	log.WithField("file", art.Name).Info("uploading")
	if err := p.uploadToS3(ctx, upload, art); err != nil {
		return fmt.Errorf("could not upload file: %w", err)
	}

	item, err := p.createOrUpdate(ctx, cfg, name, upload.FileKey)
	if err != nil {
		return fmt.Errorf("could not save custom app: %w", err)
	}

	log.
		WithField("name", item.Name).
		WithField("id", item.ID).
		Info("published custom app to iru")
	return nil
}

type uploadDetails struct {
	PostURL  string            `json:"post_url"`
	PostData map[string]string `json:"post_data"`
	FileKey  string            `json:"file_key"`
}

// initUpload asks the Iru API for pre-signed S3 upload details.
func (p Pipe) initUpload(ctx *context.Context, cfg config.Iru, fileName string) (*uploadDetails, error) {
	body, err := json.Marshal(map[string]string{"name": fileName})
	if err != nil {
		return nil, err
	}

	var details uploadDetails
	if err := retryx.Do(ctx, ctx.Config.Retry, func() error {
		return p.apiDo(ctx, cfg, http.MethodPost, cfg.URL+"/api/v1/library/custom-apps/upload", "application/json", body, &details)
	}, retryx.IsRetriable); err != nil {
		return nil, err
	}
	if details.PostURL == "" || details.FileKey == "" {
		return nil, errors.New("invalid upload response: missing post_url or file_key")
	}
	return &details, nil
}

// uploadToS3 posts the file to the pre-signed S3 URL, sending all fields
// from post_data followed by the file itself.
func (Pipe) uploadToS3(ctx *context.Context, details *uploadDetails, art *artifact.Artifact) error {
	info, err := os.Stat(art.Path)
	if err != nil {
		return err
	}

	return retryx.Do(ctx, ctx.Config.Retry, func() error {
		file, err := os.Open(art.Path)
		if err != nil {
			return retryx.Unrecoverable(err)
		}
		defer file.Close()

		var head bytes.Buffer
		writer := multipart.NewWriter(&head)
		for key, value := range details.PostData {
			if err := writer.WriteField(key, value); err != nil {
				return retryx.Unrecoverable(err)
			}
		}
		// S3 requires the file to be the last field in the form. The part
		// header lands in head, the content is streamed from the file, and
		// the closing boundary follows as the tail.
		if _, err := writer.CreateFormFile("file", art.Name); err != nil {
			return retryx.Unrecoverable(err)
		}
		tail := "\r\n--" + writer.Boundary() + "--\r\n"

		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			details.PostURL,
			io.MultiReader(bytes.NewReader(head.Bytes()), file, strings.NewReader(tail)),
		)
		if err != nil {
			return retryx.Unrecoverable(err)
		}
		req.ContentLength = int64(head.Len()) + info.Size() + int64(len(tail))
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return retryx.HTTP(err, resp)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			respBody, _ := io.ReadAll(resp.Body)
			return retryx.HTTP(fmt.Errorf("got status code %d: %s", resp.StatusCode, string(respBody)), resp)
		}
		return nil
	}, retryx.IsRetriable)
}

type customApp struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// createOrUpdate creates a new Custom App library item, or, if
// library_item_id is set, updates the existing one to point at the newly
// uploaded file. Optional fields are only sent when configured, so updates
// do not reset settings managed in the Iru dashboard.
func (p Pipe) createOrUpdate(ctx *context.Context, cfg config.Iru, name, fileKey string) (*customApp, error) {
	form := url.Values{}
	form.Set("name", name)
	form.Set("file_key", fileKey)
	form.Set("install_type", cfg.InstallType)
	form.Set("install_enforcement", cfg.InstallEnforcement)
	for key, value := range map[string]string{
		"unzip_location":           cfg.UnzipLocation,
		"audit_script":             cfg.AuditScript,
		"preinstall_script":        cfg.PreinstallScript,
		"postinstall_script":       cfg.PostinstallScript,
		"self_service_category_id": cfg.SelfServiceCategoryID,
	} {
		if value != "" {
			form.Set(key, value)
		}
	}
	for key, value := range map[string]*bool{
		"show_in_self_service":     cfg.ShowInSelfService,
		"self_service_recommended": cfg.SelfServiceRecommended,
		"restart":                  cfg.Restart,
	} {
		if value != nil {
			form.Set(key, strconv.FormatBool(*value))
		}
	}
	body := []byte(form.Encode())

	var item customApp
	if cfg.LibraryItemID != "" {
		endpoint := cfg.URL + "/api/v1/library/custom-apps/" + url.PathEscape(cfg.LibraryItemID)
		if err := retryx.Do(ctx, ctx.Config.Retry, func() error {
			return p.apiDo(ctx, cfg, http.MethodPatch, endpoint, "application/x-www-form-urlencoded", body, &item)
		}, retryx.IsRetriable); err != nil {
			return nil, err
		}
		return &item, nil
	}

	// Creating is not idempotent: a retry whose previous attempt did reach
	// the server would create a duplicate library item. Only clean
	// rejections are retried: right after the S3 upload the API answers
	// with a 503 ("The upload is still being processed") until the file
	// has been ingested, and a 503/429 guarantees nothing was created.
	if err := retryx.Do(ctx, ctx.Config.Retry, func() error {
		return p.apiDo(ctx, cfg, http.MethodPost, cfg.URL+"/api/v1/library/custom-apps", "application/x-www-form-urlencoded", body, &item)
	}, isCleanRejection); err != nil {
		return nil, err
	}
	return &item, nil
}

// isCleanRejection returns true for errors where the server definitely did
// not process the request, making a retry of a non-idempotent call safe.
func isCleanRejection(err error) bool {
	if he, ok := errors.AsType[retryx.HTTPError](err); ok {
		return he.Status == http.StatusServiceUnavailable ||
			he.Status == http.StatusTooManyRequests
	}
	return false
}

// apiDo performs an authenticated request against the Iru API and decodes
// the JSON response into out.
func (Pipe) apiDo(ctx *context.Context, cfg config.Iru, method, endpoint, contentType string, body []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return retryx.Unrecoverable(err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+cfg.APIToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return retryx.HTTP(err, resp)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return retryx.HTTP(fmt.Errorf("got status code %d: %s", resp.StatusCode, string(respBody)), resp)
	}
	return json.Unmarshal(respBody, out)
}
