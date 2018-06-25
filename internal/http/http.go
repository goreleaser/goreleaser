// Package http implements functionality common to HTTP uploading pipelines.
package http

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	h "net/http"
	"net/url"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pipeline"
)

const (
	// ModeBinary uploads only compiled binaries
	ModeBinary = "binary"
	// ModeArchive uploads release archives
	ModeArchive = "archive"
)

// Defaults sets default configuration options on Put structs
func Defaults(puts []config.Put) error {
	for i := range puts {
		defaults(&puts[i])
	}
	return nil
}

func defaults(put *config.Put) {
	if put.Mode == "" {
		put.Mode = ModeArchive
	}
}

// CheckConfig validates an HTTPUpload configuration returning a descriptive error when appropriate
func CheckConfig(ctx *context.Context, upload *config.Put, kind string) error {

	if upload.Target == "" {
		return misconfigured(kind, upload, "missing target")
	}

	if upload.Username == "" {
		return misconfigured(kind, upload, "missing username")
	}

	if upload.Name == "" {
		return misconfigured(kind, upload, "missing name")
	}

	envName := fmt.Sprintf("%s_%s_SECRET", strings.ToUpper(kind), strings.ToUpper(upload.Name))
	if _, ok := ctx.Env[envName]; !ok {
		return misconfigured(kind, upload, fmt.Sprintf("missing %s environment variable", envName))
	}

	return nil

}

func misconfigured(kind string, upload *config.Put, reason string) error {
	return pipeline.Skip(fmt.Sprintf("%s section '%s' is not configured properly (%s)", kind, upload.Name, reason))
}

// ResponseChecker is a function capable of validating an http server response.
// It must return the location of the uploaded asset or the error when the
// response must be considered a failure.
type ResponseChecker func(*h.Response) (string, error)

// Upload does the actual uploading work
func Upload(ctx *context.Context, puts []config.Put, kind string, check ResponseChecker) error {
	if ctx.SkipPublish {
		return pipeline.ErrSkipPublishEnabled
	}

	// Handle every configured put
	for _, put := range puts {
		filters := []artifact.Filter{}
		if put.Checksum {
			filters = append(filters, artifact.ByType(artifact.Checksum))
		}
		if put.Signature {
			filters = append(filters, artifact.ByType(artifact.Signature))
		}
		// We support two different modes
		//	- "archive": Upload all artifacts
		//	- "binary": Upload only the raw binaries
		switch v := strings.ToLower(put.Mode); v {
		case ModeArchive:
			filters = append(filters,
				artifact.ByType(artifact.UploadableArchive),
				artifact.ByType(artifact.LinuxPackage))
		case ModeBinary:
			filters = append(filters,
				artifact.ByType(artifact.UploadableBinary))
		default:
			err := fmt.Errorf("%s: mode \"%s\" not supported", kind, v)
			log.WithFields(log.Fields{
				kind:   put.Name,
				"mode": v,
			}).Error(err.Error())
			return err
		}
		if err := runPipeByFilter(ctx, put, artifact.Or(filters...), kind, check); err != nil {
			return err
		}
	}

	return nil
}

func runPipeByFilter(ctx *context.Context, instance config.Put, filter artifact.Filter, kind string, check ResponseChecker) error {
	sem := make(chan bool, ctx.Parallelism)
	var g errgroup.Group
	for _, artifact := range ctx.Artifacts.Filter(filter).List() {
		sem <- true
		artifact := artifact
		g.Go(func() error {
			defer func() {
				<-sem
			}()
			return uploadAsset(ctx, instance, artifact, kind, check)
		})
	}
	return g.Wait()
}

// uploadAsset uploads file to target and logs all actions
func uploadAsset(ctx *context.Context, instance config.Put, artifact artifact.Artifact, kind string, check ResponseChecker) error {
	envName := fmt.Sprintf("%s_%s_SECRET", strings.ToUpper(kind), strings.ToUpper(instance.Name))
	secret := ctx.Env[envName]

	// Generate the target url
	targetURL, err := resolveTargetTemplate(ctx, instance, artifact)
	if err != nil {
		msg := fmt.Sprintf("%s: error while building the target url", kind)
		log.WithField("instance", instance.Name).WithError(err).Error(msg)
		return errors.Wrap(err, msg)
	}

	// Handle the artifact
	file, err := os.Open(artifact.Path)
	if err != nil {
		return err
	}
	defer file.Close() // nolint: errcheck

	// The target url needs to contain the artifact name
	if !strings.HasSuffix(targetURL, "/") {
		targetURL += "/"
	}
	targetURL += artifact.Name

	location, _, err := uploadAssetToServer(ctx, targetURL, instance.Username, secret, file, check)
	if err != nil {
		msg := fmt.Sprintf("%s: upload failed", kind)
		log.WithError(err).WithFields(log.Fields{
			"instance": instance.Name,
			"username": instance.Username,
		}).Error(msg)
		return errors.Wrap(err, msg)
	}

	log.WithFields(log.Fields{
		"instance": instance.Name,
		"mode":     instance.Mode,
		"uri":      location,
	}).Info("uploaded successful")

	return nil
}

// uploadAssetToServer uploads the asset file to target
func uploadAssetToServer(ctx *context.Context, target, username, secret string, file *os.File, check ResponseChecker) (string, *h.Response, error) {
	stat, err := file.Stat()
	if err != nil {
		return "", nil, err
	}
	if stat.IsDir() {
		return "", nil, errors.New("the asset to upload can't be a directory")
	}

	req, err := newUploadRequest(target, username, secret, file, stat.Size())
	if err != nil {
		return "", nil, err
	}

	loc, resp, err := executeHTTPRequest(ctx, req, check)
	if err != nil {
		return "", resp, err
	}
	return loc, resp, nil
}

// newUploadRequest creates a new h.Request for uploading
func newUploadRequest(target, username, secret string, reader io.Reader, size int64) (*h.Request, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	req, err := h.NewRequest("PUT", u.String(), reader)
	if err != nil {
		return nil, err
	}

	req.ContentLength = size
	req.SetBasicAuth(username, secret)

	return req, err
}

// executeHTTPRequest processes the http call with respect of context ctx
func executeHTTPRequest(ctx *context.Context, req *h.Request, check ResponseChecker) (string, *h.Response, error) {
	resp, err := h.DefaultClient.Do(req)
	if err != nil {
		// If we got an error, and the context has been canceled,
		// the context's error is probably more useful.
		select {
		case <-ctx.Done():
			return "", nil, ctx.Err()
		default:
		}

		return "", nil, err
	}

	defer resp.Body.Close() // nolint: errcheck

	loc, err := check(resp)
	if err != nil {
		// even though there was an error, we still return the response
		// in case the caller wants to inspect it further
		return "", resp, err
	}

	return loc, resp, err
}

// targetData is used as a template struct for
// Artifactory.Target
type targetData struct {
	Version     string
	Tag         string
	ProjectName string

	// Only supported in mode binary
	Os   string
	Arch string
	Arm  string
}

// resolveTargetTemplate returns the resolved target template with replaced variables
// Those variables can be replaced by the given context, goos, goarch, goarm and more
func resolveTargetTemplate(ctx *context.Context, artifactory config.Put, artifact artifact.Artifact) (string, error) {
	data := targetData{
		Version:     ctx.Version,
		Tag:         ctx.Git.CurrentTag,
		ProjectName: ctx.Config.ProjectName,
	}

	if artifactory.Mode == ModeBinary {
		data.Os = replace(ctx.Config.Archive.Replacements, artifact.Goos)
		data.Arch = replace(ctx.Config.Archive.Replacements, artifact.Goarch)
		data.Arm = replace(ctx.Config.Archive.Replacements, artifact.Goarm)
	}

	var out bytes.Buffer
	t, err := template.New(ctx.Config.ProjectName).Parse(artifactory.Target)
	if err != nil {
		return "", err
	}
	err = t.Execute(&out, data)
	return out.String(), err
}

func replace(replacements map[string]string, original string) string {
	result := replacements[original]
	if result == "" {
		return original
	}
	return result
}
