package http

import (
	"bytes"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	h "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestAssetOpenDefault(t *testing.T) {
	tf := filepath.Join(t.TempDir(), "asset")
	require.NoError(t, os.WriteFile(tf, []byte("a"), 0o765))

	a, err := assetOpenDefault("blah", &artifact.Artifact{
		Path: tf,
	})
	if err != nil {
		t.Fatalf("can not open asset: %v", err)
	}
	t.Cleanup(func() {
		require.NoError(t, a.ReadCloser.Close())
	})
	bs, err := io.ReadAll(a.ReadCloser)
	if err != nil {
		t.Fatalf("can not read asset: %v", err)
	}
	if string(bs) != "a" {
		t.Fatalf("unexpected read content")
	}
	_, err = assetOpenDefault("blah", &artifact.Artifact{
		Path: "blah",
	})
	if err == nil {
		t.Fatalf("should fail on missing file")
	}
	_, err = assetOpenDefault("blah", &artifact.Artifact{
		Path: os.TempDir(),
	})
	if err == nil {
		t.Fatalf("should fail on existing dir")
	}
}

func TestDefaults(t *testing.T) {
	type args struct {
		uploads []config.Upload
	}
	tests := []struct {
		name     string
		args     args
		wantErr  bool
		wantMode string
	}{
		{"set default", args{[]config.Upload{{Name: "a", Target: "http://"}}}, false, ModeArchive},
		{"keep value", args{[]config.Upload{{Name: "a", Target: "http://...", Mode: ModeBinary}}}, false, ModeBinary},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Defaults(tt.args.uploads); (err != nil) != tt.wantErr {
				t.Errorf("Defaults() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantMode != tt.args.uploads[0].Mode {
				t.Errorf("Incorrect Defaults() mode %q , wanted %q", tt.args.uploads[0].Mode, tt.wantMode)
			}
		})
	}
}

func TestCheckConfig(t *testing.T) {
	ctx := context.New(config.Project{ProjectName: "blah"})
	ctx.Env["TEST_A_SECRET"] = "x"
	type args struct {
		ctx    *context.Context
		upload *config.Upload
		kind   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"ok", args{ctx, &config.Upload{Name: "a", Target: "http://blabla", Username: "pepe", Mode: ModeArchive}, "test"}, false},
		{"secret missing", args{ctx, &config.Upload{Name: "b", Target: "http://blabla", Username: "pepe", Mode: ModeArchive}, "test"}, true},
		{"target missing", args{ctx, &config.Upload{Name: "a", Username: "pepe", Mode: ModeArchive}, "test"}, true},
		{"name missing", args{ctx, &config.Upload{Target: "http://blabla", Username: "pepe", Mode: ModeArchive}, "test"}, true},
		{"username missing", args{ctx, &config.Upload{Name: "a", Target: "http://blabla", Mode: ModeArchive}, "test"}, true},
		{"username present", args{ctx, &config.Upload{Name: "a", Target: "http://blabla", Username: "pepe", Mode: ModeArchive}, "test"}, false},
		{"mode missing", args{ctx, &config.Upload{Name: "a", Target: "http://blabla", Username: "pepe"}, "test"}, true},
		{"mode invalid", args{ctx, &config.Upload{Name: "a", Target: "http://blabla", Username: "pepe", Mode: "blabla"}, "test"}, true},
		{"cert invalid", args{ctx, &config.Upload{Name: "a", Target: "http://blabla", Username: "pepe", Mode: ModeBinary, TrustedCerts: "bad cert!"}, "test"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckConfig(tt.args.ctx, tt.args.upload, tt.args.kind); (err != nil) != tt.wantErr {
				t.Errorf("CheckConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	delete(ctx.Env, "TEST_A_SECRET")

	tests = []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"username missing", args{ctx, &config.Upload{Name: "a", Target: "http://blabla", Mode: ModeArchive}, "test"}, false},
		{"username present", args{ctx, &config.Upload{Name: "a", Target: "http://blabla", Username: "pepe", Mode: ModeArchive}, "test"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckConfig(tt.args.ctx, tt.args.upload, tt.args.kind); (err != nil) != tt.wantErr {
				t.Errorf("CheckConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type check struct {
	path    string
	user    string
	pass    string
	content []byte
	headers map[string]string
}

func checks(checks ...check) func(rs []*h.Request) error {
	return func(rs []*h.Request) error {
		if len(rs) != len(checks) {
			return fmt.Errorf("expected %d requests, got %d", len(checks), len(rs))
		}
		for _, r := range rs {
			found := false
			for _, c := range checks {
				if c.path == r.RequestURI {
					found = true
					err := doCheck(c, r)
					if err != nil {
						return err
					}
					break
				}
			}
			if !found {
				return fmt.Errorf("check not found for request %+v", r)
			}
		}
		return nil
	}
}

func doCheck(c check, r *h.Request) error {
	contentLength := int64(len(c.content))
	if r.ContentLength != contentLength {
		return fmt.Errorf("request content-length header value %v unexpected, wanted %v", r.ContentLength, contentLength)
	}
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("reading request body: %v", err)
	}
	if !bytes.Equal(bs, c.content) {
		return errors.New("content does not match")
	}
	if int64(len(bs)) != contentLength {
		return fmt.Errorf("request content length %v unexpected, wanted %v", int64(len(bs)), contentLength)
	}
	if r.RequestURI != c.path {
		return fmt.Errorf("bad request uri %q, expecting %q", r.RequestURI, c.path)
	}
	if u, p, ok := r.BasicAuth(); !ok || u != c.user || p != c.pass {
		return fmt.Errorf("bad basic auth credentials: %s/%s", u, p)
	}
	for k, v := range c.headers {
		if r.Header.Get(k) != v {
			return fmt.Errorf("bad header value for %s: expected %s, got %s", k, v, r.Header.Get(k))
		}
	}
	return nil
}

func TestUpload(t *testing.T) {
	content := []byte("blah!")
	requests := []*h.Request{}
	var m sync.Mutex
	mux := h.NewServeMux()
	mux.Handle("/", h.HandlerFunc(func(w h.ResponseWriter, r *h.Request) {
		bs, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(h.StatusInternalServerError)
			fmt.Fprintf(w, "reading request body: %v", err)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(bs))
		m.Lock()
		requests = append(requests, r)
		m.Unlock()
		w.WriteHeader(h.StatusCreated)
		w.Header().Set("Location", r.URL.RequestURI())
	}))
	assetOpen = func(k string, a *artifact.Artifact) (*asset, error) {
		return &asset{
			ReadCloser: io.NopCloser(bytes.NewReader(content)),
			Size:       int64(len(content)),
		}, nil
	}
	defer assetOpenReset()
	var is2xx ResponseChecker = func(r *h.Response) error {
		if r.StatusCode/100 == 2 {
			return nil
		}
		return fmt.Errorf("unexpected http status code: %v", r.StatusCode)
	}
	ctx := context.New(config.Project{
		ProjectName: "blah",
		Archives: []config.Archive{
			{
				Replacements: map[string]string{
					"linux": "Linux",
				},
			},
		},
	})
	ctx.Env["TEST_A_SECRET"] = "x"
	ctx.Env["TEST_A_USERNAME"] = "u2"
	ctx.Version = "2.1.0"
	ctx.Artifacts = artifact.New()
	folder := t.TempDir()
	for _, a := range []struct {
		ext string
		typ artifact.Type
	}{
		{"---", artifact.DockerImage},
		{"deb", artifact.LinuxPackage},
		{"bin", artifact.Binary},
		{"tar", artifact.UploadableArchive},
		{"ubi", artifact.UploadableBinary},
		{"sum", artifact.Checksum},
		{"sig", artifact.Signature},
	} {
		file := filepath.Join(folder, "a."+a.ext)
		require.NoError(t, os.WriteFile(file, []byte("lorem ipsum"), 0o644))
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   "a." + a.ext,
			Goos:   "linux",
			Goarch: "amd64",
			Path:   file,
			Type:   a.typ,
			Extra: map[string]interface{}{
				"ID": "foo",
			},
		})
	}

	tests := []struct {
		name         string
		tryPlain     bool
		tryTLS       bool
		wantErrPlain bool
		wantErrTLS   bool
		setup        func(*httptest.Server) (*context.Context, config.Upload)
		check        func(r []*h.Request) error
	}{
		{
			"wrong-mode", true, true, true, true,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				return ctx, config.Upload{
					Mode:         "wrong-mode",
					Name:         "a",
					Target:       s.URL + "/{{.ProjectName}}/{{.Version}}/",
					Username:     "u1",
					TrustedCerts: cert(s),
				}
			},
			checks(),
		},
		{
			"username-from-env", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				return ctx, config.Upload{
					Mode:         ModeArchive,
					Name:         "a",
					Target:       s.URL + "/{{.ProjectName}}/{{.Version}}/",
					TrustedCerts: cert(s),
				}
			},
			checks(
				check{"/blah/2.1.0/a.deb", "u2", "x", content, map[string]string{}},
				check{"/blah/2.1.0/a.tar", "u2", "x", content, map[string]string{}},
			),
		},
		{
			"post", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				return ctx, config.Upload{
					Method:       h.MethodPost,
					Mode:         ModeArchive,
					Name:         "a",
					Target:       s.URL + "/{{.ProjectName}}/{{.Version}}/",
					Username:     "u1",
					TrustedCerts: cert(s),
				}
			},
			checks(
				check{"/blah/2.1.0/a.deb", "u1", "x", content, map[string]string{}},
				check{"/blah/2.1.0/a.tar", "u1", "x", content, map[string]string{}},
			),
		},
		{
			"archive", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				return ctx, config.Upload{
					Mode:         ModeArchive,
					Name:         "a",
					Target:       s.URL + "/{{.ProjectName}}/{{.Version}}/",
					Username:     "u1",
					TrustedCerts: cert(s),
				}
			},
			checks(
				check{"/blah/2.1.0/a.deb", "u1", "x", content, map[string]string{}},
				check{"/blah/2.1.0/a.tar", "u1", "x", content, map[string]string{}},
			),
		},
		{
			"archive_with_os_tmpl", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				return ctx, config.Upload{
					Mode:         ModeArchive,
					Name:         "a",
					Target:       s.URL + "/{{.ProjectName}}/{{.Version}}/{{.Os}}/{{.Arch}}",
					Username:     "u1",
					TrustedCerts: cert(s),
				}
			},
			checks(
				check{"/blah/2.1.0/linux/amd64/a.deb", "u1", "x", content, map[string]string{}},
				check{"/blah/2.1.0/linux/amd64/a.tar", "u1", "x", content, map[string]string{}},
			),
		},
		{
			"archive_with_ids", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				return ctx, config.Upload{
					Mode:         ModeArchive,
					Name:         "a",
					Target:       s.URL + "/{{.ProjectName}}/{{.Version}}/",
					Username:     "u1",
					TrustedCerts: cert(s),
					IDs:          []string{"foo"},
				}
			},
			checks(
				check{"/blah/2.1.0/a.deb", "u1", "x", content, map[string]string{}},
				check{"/blah/2.1.0/a.tar", "u1", "x", content, map[string]string{}},
			),
		},
		{
			"binary", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				return ctx, config.Upload{
					Mode:         ModeBinary,
					Name:         "a",
					Target:       s.URL + "/{{.ProjectName}}/{{.Version}}/",
					Username:     "u2",
					TrustedCerts: cert(s),
				}
			},
			checks(check{"/blah/2.1.0/a.ubi", "u2", "x", content, map[string]string{}}),
		},
		{
			"binary_with_os_tmpl", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				return ctx, config.Upload{
					Mode:         ModeBinary,
					Name:         "a",
					Target:       s.URL + "/{{.ProjectName}}/{{.Version}}/{{.Os}}/{{.Arch}}",
					Username:     "u2",
					TrustedCerts: cert(s),
				}
			},
			checks(check{"/blah/2.1.0/Linux/amd64/a.ubi", "u2", "x", content, map[string]string{}}),
		},
		{
			"binary_with_ids", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				return ctx, config.Upload{
					Mode:         ModeBinary,
					Name:         "a",
					Target:       s.URL + "/{{.ProjectName}}/{{.Version}}/",
					Username:     "u2",
					TrustedCerts: cert(s),
					IDs:          []string{"foo"},
				}
			},
			checks(check{"/blah/2.1.0/a.ubi", "u2", "x", content, map[string]string{}}),
		},
		{
			"binary-add-ending-bar", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				return ctx, config.Upload{
					Mode:         ModeBinary,
					Name:         "a",
					Target:       s.URL + "/{{.ProjectName}}/{{.Version}}",
					Username:     "u2",
					TrustedCerts: cert(s),
				}
			},
			checks(check{"/blah/2.1.0/a.ubi", "u2", "x", content, map[string]string{}}),
		},
		{
			"archive-with-checksum-and-signature", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				return ctx, config.Upload{
					Mode:         ModeArchive,
					Name:         "a",
					Target:       s.URL + "/{{.ProjectName}}/{{.Version}}/",
					Username:     "u3",
					Checksum:     true,
					Signature:    true,
					TrustedCerts: cert(s),
				}
			},
			checks(
				check{"/blah/2.1.0/a.deb", "u3", "x", content, map[string]string{}},
				check{"/blah/2.1.0/a.tar", "u3", "x", content, map[string]string{}},
				check{"/blah/2.1.0/a.sum", "u3", "x", content, map[string]string{}},
				check{"/blah/2.1.0/a.sig", "u3", "x", content, map[string]string{}},
			),
		},
		{
			"bad-template", true, true, true, true,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				return ctx, config.Upload{
					Mode:         ModeBinary,
					Name:         "a",
					Target:       s.URL + "/{{.ProjectNameXXX}}/{{.VersionXXX}}/",
					Username:     "u3",
					Checksum:     true,
					Signature:    true,
					TrustedCerts: cert(s),
				}
			},
			checks(),
		},
		{
			"failed-request", true, true, true, true,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				return ctx, config.Upload{
					Mode:         ModeBinary,
					Name:         "a",
					Target:       s.URL[0:strings.LastIndex(s.URL, ":")] + "/{{.ProjectName}}/{{.Version}}/",
					Username:     "u3",
					Checksum:     true,
					Signature:    true,
					TrustedCerts: cert(s),
				}
			},
			checks(),
		},
		{
			"broken-cert", false, true, false, true,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				return ctx, config.Upload{
					Mode:         ModeBinary,
					Name:         "a",
					Target:       s.URL + "/{{.ProjectName}}/{{.Version}}/",
					Username:     "u3",
					Checksum:     false,
					Signature:    false,
					TrustedCerts: "bad certs!",
				}
			},
			checks(),
		},
		{
			"skip-publishing", true, true, true, true,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				c := *ctx
				c.SkipPublish = true
				return &c, config.Upload{}
			},
			checks(),
		},
		{
			"checksumheader", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				return ctx, config.Upload{
					Mode:           ModeBinary,
					Name:           "a",
					Target:         s.URL + "/{{.ProjectName}}/{{.Version}}/",
					Username:       "u2",
					ChecksumHeader: "-x-sha256",
					TrustedCerts:   cert(s),
				}
			},
			checks(check{"/blah/2.1.0/a.ubi", "u2", "x", content, map[string]string{"-x-sha256": "5e2bf57d3f40c4b6df69daf1936cb766f832374b4fc0259a7cbff06e2f70f269"}}),
		},
		{
			"custom-headers", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				return ctx, config.Upload{
					Mode:     ModeBinary,
					Name:     "a",
					Target:   s.URL + "/{{.ProjectName}}/{{.Version}}/",
					Username: "u2",
					CustomHeaders: map[string]string{
						"x-custom-header-name": "custom-header-value",
					},
					TrustedCerts: cert(s),
				}
			},
			checks(check{"/blah/2.1.0/a.ubi", "u2", "x", content, map[string]string{"x-custom-header-name": "custom-header-value"}}),
		},
		{
			"custom-headers-with-template", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				return ctx, config.Upload{
					Mode:     ModeBinary,
					Name:     "a",
					Target:   s.URL + "/{{.ProjectName}}/{{.Version}}/",
					Username: "u2",
					CustomHeaders: map[string]string{
						"x-project-name": "{{ .ProjectName }}",
					},
					TrustedCerts: cert(s),
				}
			},
			checks(check{"/blah/2.1.0/a.ubi", "u2", "x", content, map[string]string{"x-project-name": "blah"}}),
		},
		{
			"invalid-template-in-custom-headers", true, true, true, true,
			func(s *httptest.Server) (*context.Context, config.Upload) {
				return ctx, config.Upload{
					Mode:     ModeBinary,
					Name:     "a",
					Target:   s.URL + "/{{.ProjectName}}/{{.Version}}/",
					Username: "u2",
					CustomHeaders: map[string]string{
						"x-custom-header-name": "{{ .Env.NONEXISTINGVARIABLE and some bad expressions }}",
					},
					TrustedCerts: cert(s),
				}
			},
			checks(),
		},
	}

	uploadAndCheck := func(t *testing.T, setup func(*httptest.Server) (*context.Context, config.Upload), wantErrPlain, wantErrTLS bool, check func(r []*h.Request) error, srv *httptest.Server) {
		t.Helper()
		requests = nil
		ctx, upload := setup(srv)
		wantErr := wantErrPlain
		if srv.Certificate() != nil {
			wantErr = wantErrTLS
		}
		if err := Upload(ctx, []config.Upload{upload}, "test", is2xx); (err != nil) != wantErr {
			t.Errorf("Upload() error = %v, wantErr %v", err, wantErr)
		}
		if err := check(requests); err != nil {
			t.Errorf("Upload() request invalid. Error: %v", err)
		}
	}

	for _, tt := range tests {
		if tt.tryPlain {
			t.Run(tt.name, func(t *testing.T) {
				srv := httptest.NewServer(mux)
				defer srv.Close()
				uploadAndCheck(t, tt.setup, tt.wantErrPlain, tt.wantErrTLS, tt.check, srv)
			})
		}
		if tt.tryTLS {
			t.Run(tt.name+"-tls", func(t *testing.T) {
				srv := httptest.NewUnstartedServer(mux)
				srv.StartTLS()
				defer srv.Close()
				uploadAndCheck(t, tt.setup, tt.wantErrPlain, tt.wantErrTLS, tt.check, srv)
			})
		}
	}
}

func cert(srv *httptest.Server) string {
	if srv == nil || srv.Certificate() == nil {
		return ""
	}
	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: srv.Certificate().Raw,
	}
	return string(pem.EncodeToMemory(block))
}
