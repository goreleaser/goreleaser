// Package http implements functionality common to HTTP uploading pipelines.
package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	h "net/http"
	"os"
	"runtime"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	// ModeBinary uploads only compiled binaries.
	ModeBinary = "binary"
	// ModeArchive uploads release archives.
	ModeArchive = "archive"
)

type asset struct {
	ReadCloser io.ReadCloser
	Size       int64
}

type assetOpenFunc func(string, *artifact.Artifact) (*asset, error)

// nolint: gochecknoglobals
var assetOpen assetOpenFunc

// TODO: fix this.
// nolint: gochecknoinits
func init() {
	assetOpenReset()
}

func assetOpenReset() {
	assetOpen = assetOpenDefault
}

func assetOpenDefault(kind string, a *artifact.Artifact) (*asset, error) {
	f, err := os.Open(a.Path)
	if err != nil {
		return nil, err
	}
	s, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if s.IsDir() {
		return nil, fmt.Errorf("%s: upload failed: the asset to upload can't be a directory", kind)
	}
	return &asset{
		ReadCloser: f,
		Size:       s.Size(),
	}, nil
}

// Defaults sets default configuration options on upload structs.
func Defaults(uploads []config.Upload) error {
	for i := range uploads {
		defaults(&uploads[i])
	}
	return nil
}

func defaults(upload *config.Upload) {
	if upload.Mode == "" {
		upload.Mode = ModeArchive
	}
	if upload.Method == "" {
		upload.Method = h.MethodPut
	}
}

// CheckConfig validates an upload configuration returning a descriptive error when appropriate.
func CheckConfig(ctx *context.Context, upload *config.Upload, kind string) error {
	if upload.Target == "" {
		return misconfigured(kind, upload, "missing target")
	}

	if upload.Name == "" {
		return misconfigured(kind, upload, "missing name")
	}

	if upload.Mode != ModeArchive && upload.Mode != ModeBinary {
		return misconfigured(kind, upload, "mode must be 'binary' or 'archive'")
	}

	username := getUsername(ctx, upload, kind)
	password := getPassword(ctx, upload, kind)
	passwordEnv := fmt.Sprintf("%s_%s_SECRET", strings.ToUpper(kind), strings.ToUpper(upload.Name))

	if password != "" && username == "" {
		return misconfigured(kind, upload, fmt.Sprintf("'username' is required when '%s' environment variable is set", passwordEnv))
	}

	if username != "" && password == "" {
		return misconfigured(kind, upload, fmt.Sprintf("environment variable '%s' is required when 'username' is set", passwordEnv))
	}

	if upload.TrustedCerts != "" && !x509.NewCertPool().AppendCertsFromPEM([]byte(upload.TrustedCerts)) {
		return misconfigured(kind, upload, "no certificate could be added from the specified trusted_certificates configuration")
	}

	if upload.ClientX509Cert != "" && upload.ClientX509Key == "" {
		return misconfigured(kind, upload, "'client_x509_key' must be set when 'client_x509_cert' is set")
	}
	if upload.ClientX509Key != "" && upload.ClientX509Cert == "" {
		return misconfigured(kind, upload, "'client_x509_cert' must be set when 'client_x509_key' is set")
	}
	if upload.ClientX509Cert != "" && upload.ClientX509Key != "" {
		if _, err := tls.LoadX509KeyPair(upload.ClientX509Cert, upload.ClientX509Key); err != nil {
			return misconfigured(kind, upload,
				"client x509 certificate could not be loaded from the specified 'client_x509_cert' and 'client_x509_key'")
		}
	}

	return nil
}

// username is optional
func getUsername(ctx *context.Context, upload *config.Upload, kind string) string {
	if upload.Username != "" {
		return upload.Username
	}

	key := fmt.Sprintf("%s_%s_USERNAME", strings.ToUpper(kind), strings.ToUpper(upload.Name))
	return ctx.Env[key]
}

// password is optional
func getPassword(ctx *context.Context, upload *config.Upload, kind string) string {
	key := fmt.Sprintf("%s_%s_SECRET", strings.ToUpper(kind), strings.ToUpper(upload.Name))
	return ctx.Env[key]
}

func misconfigured(kind string, upload *config.Upload, reason string) error {
	return pipe.Skipf("%s section '%s' is not configured properly (%s)", kind, upload.Name, reason)
}

// ResponseChecker is a function capable of validating an http server response.
// It must return and error when the response must be considered a failure.
type ResponseChecker func(*h.Response) error

// Upload does the actual uploading work.
func Upload(ctx *context.Context, uploads []config.Upload, kind string, check ResponseChecker) error {
	// Handle every configured upload
	for _, upload := range uploads {
		upload := upload
		filters := []artifact.Filter{}
		if upload.Checksum {
			filters = append(filters, artifact.ByType(artifact.Checksum))
		}
		if upload.Meta {
			filters = append(filters, artifact.ByType(artifact.Metadata))
		}
		if upload.Signature {
			filters = append(filters, artifact.ByType(artifact.Signature), artifact.ByType(artifact.Certificate))
		}
		// We support two different modes
		//	- "archive": Upload all artifacts
		//	- "binary": Upload only the raw binaries
		switch v := strings.ToLower(upload.Mode); v {
		case ModeArchive:
			filters = append(filters,
				artifact.ByType(artifact.UploadableArchive),
				artifact.ByType(artifact.UploadableSourceArchive),
				artifact.ByType(artifact.LinuxPackage),
			)
		case ModeBinary:
			filters = append(filters, artifact.ByType(artifact.UploadableBinary))
		default:
			return fmt.Errorf("%s: %s: mode \"%s\" not supported", upload.Name, kind, v)
		}

		filter := artifact.Or(filters...)
		if len(upload.IDs) > 0 {
			filter = artifact.And(filter, artifact.ByIDs(upload.IDs...))
		}
		if len(upload.Exts) > 0 {
			filter = artifact.And(filter, artifact.ByExt(upload.Exts...))
		}
		if err := uploadWithFilter(ctx, &upload, filter, kind, check); err != nil {
			return err
		}
	}

	return nil
}

func uploadWithFilter(ctx *context.Context, upload *config.Upload, filter artifact.Filter, kind string, check ResponseChecker) error {
	artifacts := ctx.Artifacts.Filter(filter).List()
	if len(artifacts) == 0 {
		log.Info("no artifacts found")
	}
	log.Debugf("will upload %d artifacts", len(artifacts))
	g := semerrgroup.New(ctx.Parallelism)
	for _, artifact := range artifacts {
		artifact := artifact
		g.Go(func() error {
			return uploadAsset(ctx, upload, artifact, kind, check)
		})
	}
	return g.Wait()
}

// uploadAsset uploads file to target and logs all actions.
func uploadAsset(ctx *context.Context, upload *config.Upload, artifact *artifact.Artifact, kind string, check ResponseChecker) error {
	// username and secret are optional since the server may not support/need
	// basic authentication always
	username := getUsername(ctx, upload, kind)
	secret := getPassword(ctx, upload, kind)

	// Generate the target url
	targetURL, err := tmpl.New(ctx).WithArtifact(artifact).Apply(upload.Target)
	if err != nil {
		return fmt.Errorf("%s: %s: error while building target URL: %w", upload.Name, kind, err)
	}

	// Handle the artifact
	asset, err := assetOpen(kind, artifact)
	if err != nil {
		return err
	}
	defer asset.ReadCloser.Close()

	// target url need to contain the artifact name unless the custom
	// artifact name is used
	if !upload.CustomArtifactName {
		if !strings.HasSuffix(targetURL, "/") {
			targetURL += "/"
		}
		targetURL += artifact.Name
	}
	log.Debugf("generated target url: %s", targetURL)

	headers := make(map[string]string, len(upload.CustomHeaders))
	for name, value := range upload.CustomHeaders {
		resolvedValue, err := tmpl.New(ctx).WithArtifact(artifact).Apply(value)
		if err != nil {
			return fmt.Errorf("%s: %s: failed to resolve custom_headers template: %w", upload.Name, kind, err)
		}
		headers[name] = resolvedValue
	}
	if upload.ChecksumHeader != "" {
		sum, err := artifact.Checksum("sha256")
		if err != nil {
			return err
		}
		headers[upload.ChecksumHeader] = sum
	}

	res, err := uploadAssetToServer(ctx, upload, targetURL, username, secret, headers, asset, check)
	if err != nil {
		return fmt.Errorf("%s: %s: upload failed: %w", upload.Name, kind, err)
	}
	if err := res.Body.Close(); err != nil {
		log.WithError(err).Warn("failed to close response body")
	}

	log.WithField("instance", upload.Name).
		WithField("mode", upload.Mode).
		Info("uploaded successful")

	return nil
}

// uploadAssetToServer uploads the asset file to target.
func uploadAssetToServer(ctx *context.Context, upload *config.Upload, target, username, secret string, headers map[string]string, a *asset, check ResponseChecker) (*h.Response, error) {
	req, err := newUploadRequest(ctx, upload.Method, target, username, secret, headers, a)
	if err != nil {
		return nil, err
	}

	return executeHTTPRequest(ctx, upload, req, check)
}

// newUploadRequest creates a new h.Request for uploading.
func newUploadRequest(ctx *context.Context, method, target, username, secret string, headers map[string]string, a *asset) (*h.Request, error) {
	req, err := h.NewRequestWithContext(ctx, method, target, a.ReadCloser)
	if err != nil {
		return nil, err
	}
	req.ContentLength = a.Size

	if username != "" && secret != "" {
		req.SetBasicAuth(username, secret)
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	return req, err
}

func getHTTPClient(upload *config.Upload) (*h.Client, error) {
	if upload.TrustedCerts == "" && upload.ClientX509Cert == "" && upload.ClientX509Key == "" {
		return h.DefaultClient, nil
	}
	transport := &h.Transport{
		Proxy:           h.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{},
	}
	if upload.TrustedCerts != "" {
		pool, err := x509.SystemCertPool()
		if err != nil {
			if runtime.GOOS == "windows" {
				// on windows ignore errors until golang issues #16736 & #18609 get fixed
				pool = x509.NewCertPool()
			} else {
				return nil, err
			}
		}
		pool.AppendCertsFromPEM([]byte(upload.TrustedCerts)) // already validated certs checked by CheckConfig
		transport.TLSClientConfig.RootCAs = pool
	}
	if upload.ClientX509Cert != "" && upload.ClientX509Key != "" {
		cert, err := tls.LoadX509KeyPair(upload.ClientX509Cert, upload.ClientX509Key)
		if err != nil {
			return nil, err
		}
		transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
	}
	return &h.Client{Transport: transport}, nil
}

// executeHTTPRequest processes the http call with respect of context ctx.
func executeHTTPRequest(ctx *context.Context, upload *config.Upload, req *h.Request, check ResponseChecker) (*h.Response, error) {
	client, err := getHTTPClient(upload)
	if err != nil {
		return nil, err
	}
	log.Debugf("executing request: %s %s (headers: %v)", req.Method, req.URL, req.Header)
	resp, err := client.Do(req)
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

	defer resp.Body.Close()

	err = check(resp)
	if err != nil {
		// even though there was an error, we still return the response
		// in case the caller wants to inspect it further
		return resp, err
	}

	return resp, err
}
