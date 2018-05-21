package httpupload

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/alecthomas/template"
	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pipeline"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

const (
	modeBinary  = "binary"
	modeArchive = "archive"
)

type httpServerResponse struct {
	Created bool
}

// Pipe for http publishing
type Pipe struct{}

// String returns the description of the pipe
func (Pipe) String() string {
	return "releasing to HTTP"
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	if len(ctx.Config.HTTPUploads) == 0 {
		return nil
	}

	// Check if a mode was set
	for i := range ctx.Config.HTTPUploads {
		if ctx.Config.HTTPUploads[i].Mode == "" {
			ctx.Config.HTTPUploads[i].Mode = modeArchive
		}
	}

	return nil
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {

	if len(ctx.Config.HTTPUploads) == 0 {
		return pipeline.Skip("http uploads section is not configured")
	}

	// Check requirements for every instance we have configured.
	// If not fulfilled, we can skip this pipeline
	for _, instance := range ctx.Config.HTTPUploads {
		if instance.Target == "" {
			return pipeline.Skip("http upload section is not configured properly (missing target)")
		}

		if instance.Username == "" {
			return pipeline.Skip("http upload section is not configured properly (missing username)")
		}

		if instance.Name == "" {
			return pipeline.Skip("http upload section is not configured properly (missing name)")
		}

		envName := fmt.Sprintf("HTTP_UPLOAD_%s_SECRET", strings.ToUpper(instance.Name))
		if _, ok := ctx.Env[envName]; !ok {
			return pipeline.Skip(fmt.Sprintf("missing secret for http upload instance %s", instance.Name))
		}
	}

	return doRun(ctx)
}

func doRun(ctx *context.Context) error {

	if ctx.SkipPublish {
		return pipeline.ErrSkipPublishEnabled
	}

	// Handle every configured http upload instance
	for _, instance := range ctx.Config.HTTPUploads {
		// We support two different modes
		//	- "archive": Upload all artifacts
		//	- "binary": Upload only the raw binaries
		var filter artifact.Filter
		switch v := strings.ToLower(instance.Mode); v {
		case modeArchive:
			filter = artifact.Or(
				artifact.ByType(artifact.UploadableArchive),
				artifact.ByType(artifact.LinuxPackage),
				artifact.ByType(artifact.Checksum),
				artifact.ByType(artifact.Signature),
			)
		case modeBinary:
			filter = artifact.ByType(artifact.UploadableBinary)
		default:
			err := fmt.Errorf("httpupload: mode \"%s\" not supported", v)
			log.WithFields(log.Fields{
				"instance": instance.Name,
				"mode":     v,
			}).Error(err.Error())
			return err
		}

		if err := runPipeByFilter(ctx, instance, filter); err != nil {
			return err
		}
	}

	return nil
}

func runPipeByFilter(ctx *context.Context, instance config.HTTPUpload, filter artifact.Filter) error {
	sem := make(chan bool, ctx.Parallelism)
	var g errgroup.Group
	for _, artifact := range ctx.Artifacts.Filter(filter).List() {
		sem <- true
		artifact := artifact
		g.Go(func() error {
			defer func() {
				<-sem
			}()
			return uploadAsset(ctx, instance, artifact)
		})
	}
	return g.Wait()
}

func uploadAsset(ctx *context.Context, instance config.HTTPUpload, artifact artifact.Artifact) error {
	envName := fmt.Sprintf("HTTP_UPLOAD_%s_SECRET", strings.ToUpper(instance.Name))
	secret := ctx.Env[envName]

	// Generate the target url
	targetURL, err := resolveTargetTemplate(ctx, instance, artifact)
	if err != nil {
		msg := "httpupload: error while building the target url"
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

	_, _, err = uploadAssetToHTTPServer(ctx, targetURL, instance.Username, secret, file)
	if err != nil {
		msg := "httpupload: upload failed"
		log.WithError(err).WithFields(log.Fields{
			"instance": instance.Name,
			"username": instance.Username,
		}).Error(msg)
		return errors.Wrap(err, msg)
	}

	log.WithFields(log.Fields{
		"instance": instance.Name,
		"mode":     instance.Mode,
		"uri":      targetURL,
	}).Info("uploaded successful")

	return nil
}

// targetData is used as a template struct for HTTPUpload.Target
type targetData struct {
	Version     string
	Tag         string
	ProjectName string

	// Only supported in mode binary
	Os   string
	Arch string
	Arm  string
}

// resolveTargetTemplate returns the resolved target template with replaced variables.
// Those variables can be replaced by the given context, goos, goarch, goarm and more.
func resolveTargetTemplate(ctx *context.Context, upload config.HTTPUpload, artifact artifact.Artifact) (string, error) {
	data := targetData{
		Version:     ctx.Version,
		Tag:         ctx.Git.CurrentTag,
		ProjectName: ctx.Config.ProjectName,
	}

	if upload.Mode == modeBinary {
		data.Os = replace(ctx.Config.Archive.Replacements, artifact.Goos)
		data.Arch = replace(ctx.Config.Archive.Replacements, artifact.Goarch)
		data.Arm = replace(ctx.Config.Archive.Replacements, artifact.Goarm)
	}

	var out bytes.Buffer
	t, err := template.New(ctx.Config.ProjectName).Parse(upload.Target)
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

// uploadAssetToHTTPServer uploads the asset file to target
func uploadAssetToHTTPServer(ctx *context.Context, target, username, secret string, file *os.File) (*httpServerResponse, *http.Response, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	if stat.IsDir() {
		return nil, nil, errors.New("the asset to upload can't be a directory")
	}

	req, err := newUploadRequest(target, username, secret, file, stat.Size())
	if err != nil {
		return nil, nil, err
	}

	asset := new(httpServerResponse)
	resp, err := executeHTTPRequest(ctx, req, asset)
	if err != nil {
		return nil, resp, err
	}
	return asset, resp, nil
}

// newUploadRequest creates a new http.Request for uploading
func newUploadRequest(target, username, secret string, reader io.Reader, size int64) (*http.Request, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("PUT", u.String(), reader)
	if err != nil {
		return nil, err
	}

	req.ContentLength = size
	req.SetBasicAuth(username, secret)

	return req, err
}

// executeHTTPRequest processes the http call with respect of context ctx
func executeHTTPRequest(ctx *context.Context, req *http.Request, v *httpServerResponse) (*http.Response, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// If we got an error, and the context has been canceled,
		// the context's error is probably more useful.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		return nil, err
	}

	defer resp.Body.Close() // nolint: errcheck

	err = checkResponse(resp)
	if err != nil {
		// even though there was an error, we still return the response
		// in case the caller wants to inspect it further
		return resp, err
	}

	v.Created = resp.StatusCode == http.StatusCreated

	return resp, err
}

// An ErrorResponse reports one or more errors caused by an API request.
type errorResponse struct {
	Response *http.Response // HTTP response that caused this error
}

func (r *errorResponse) Error() string {
	return fmt.Sprintf("%v %v: %s",
		r.Response.Request.Method,
		r.Response.Request.URL,
		r.Response.Status)
}

// checkResponse checks the HTTP response for errors, and returns them if
// present. A response is considered an error if it has a status code outside
// the 200 range.
// API error responses are expected to have no response body.
func checkResponse(r *http.Response) error {
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}
	return &errorResponse{Response: r}
}
