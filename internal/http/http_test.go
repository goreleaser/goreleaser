package http

import (
	"bytes"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	h "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

func TestAssetOpenDefault(t *testing.T) {
	tf, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("can not create tmp file: %v", err)
		return
	}
	fmt.Fprint(tf, "a")
	tf.Close()
	a, err := assetOpenDefault("blah", &artifact.Artifact{
		Path: tf.Name(),
	})
	if err != nil {
		t.Fatalf("can not open asset: %v", err)
	}
	bs, err := ioutil.ReadAll(a.ReadCloser)
	if err != nil {
		t.Fatalf("can not read asset: %v", err)
	}
	if string(bs) != "a" {
		t.Fatalf("unexpected read content")
	}
	os.Remove(tf.Name())
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
		puts []config.Put
	}
	tests := []struct {
		name     string
		args     args
		wantErr  bool
		wantMode string
	}{
		{"set default", args{[]config.Put{{Name: "a", Target: "http://"}}}, false, ModeArchive},
		{"keep value", args{[]config.Put{{Name: "a", Target: "http://...", Mode: ModeBinary}}}, false, ModeBinary},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Defaults(tt.args.puts); (err != nil) != tt.wantErr {
				t.Errorf("Defaults() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantMode != tt.args.puts[0].Mode {
				t.Errorf("Incorrect Defaults() mode %q , wanted %q", tt.args.puts[0].Mode, tt.wantMode)
			}
		})
	}
}

func TestCheckConfig(t *testing.T) {
	ctx := context.New(config.Project{ProjectName: "blah"})
	ctx.Env["TEST_A_SECRET"] = "x"
	type args struct {
		ctx    *context.Context
		upload *config.Put
		kind   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"ok", args{ctx, &config.Put{Name: "a", Target: "http://blabla", Username: "pepe", Mode: ModeArchive}, "test"}, false},
		{"secret missing", args{ctx, &config.Put{Name: "b", Target: "http://blabla", Username: "pepe", Mode: ModeArchive}, "test"}, true},
		{"target missing", args{ctx, &config.Put{Name: "a", Username: "pepe", Mode: ModeArchive}, "test"}, true},
		{"name missing", args{ctx, &config.Put{Target: "http://blabla", Username: "pepe", Mode: ModeArchive}, "test"}, true},
		{"mode missing", args{ctx, &config.Put{Name: "a", Target: "http://blabla", Username: "pepe"}, "test"}, true},
		{"mode invalid", args{ctx, &config.Put{Name: "a", Target: "http://blabla", Username: "pepe", Mode: "blabla"}, "test"}, true},
		{"cert invalid", args{ctx, &config.Put{Name: "a", Target: "http://blabla", Username: "pepe", Mode: ModeBinary, TrustedCerts: "bad cert!"}, "test"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckConfig(tt.args.ctx, tt.args.upload, tt.args.kind); (err != nil) != tt.wantErr {
				t.Errorf("CheckConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func count(r io.Reader) (int64, error) {
	var (
		c   int64
		b   int64
		err error
		buf = make([]byte, 16)
	)
	for b >= 0 && err == nil {
		b, err := r.Read(buf)
		if err != nil {
			return c, err
		}
		c = c + int64(b)
	}
	return c, nil
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
			return errors.New("expectations mismatch requests")
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
				return errors.Errorf("check not found for request %+v", r)
			}
		}
		return nil
	}
}

func doCheck(c check, r *h.Request) error {
	contentLength := int64(len(c.content))
	if r.ContentLength != contentLength {
		return errors.Errorf("request content-length header value %v unexpected, wanted %v", r.ContentLength, contentLength)
	}
	bs, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.Errorf("reading request body: %v", err)
	}
	if !bytes.Equal(bs, c.content) {
		return errors.New("content does not match")
	}
	if int64(len(bs)) != contentLength {
		return errors.Errorf("request content length %v unexpected, wanted %v", int64(len(bs)), contentLength)
	}
	if r.RequestURI != c.path {
		return errors.Errorf("bad request uri %q, expecting %q", r.RequestURI, c.path)
	}
	if u, p, ok := r.BasicAuth(); !ok || u != c.user || p != c.pass {
		return errors.Errorf("bad basic auth credentials: %s/%s", u, p)
	}
	for k, v := range c.headers {
		if r.Header.Get(k) != v {
			return errors.Errorf("bad header value for %s: expected %s, got %s", k, v, r.Header.Get(k))
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
		bs, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(h.StatusInternalServerError)
			fmt.Fprintf(w, "reading request body: %v", err)
			return
		}
		r.Body = ioutil.NopCloser(bytes.NewReader(bs))
		m.Lock()
		requests = append(requests, r)
		m.Unlock()
		w.WriteHeader(h.StatusCreated)
		w.Header().Set("Location", r.URL.RequestURI())
	}))
	assetOpen = func(k string, a *artifact.Artifact) (*asset, error) {
		return &asset{
			ReadCloser: ioutil.NopCloser(bytes.NewReader(content)),
			Size:       int64(len(content)),
		}, nil
	}
	defer assetOpenReset()
	var is2xx ResponseChecker = func(r *h.Response) error {
		if r.StatusCode/100 == 2 {
			return nil
		}
		return errors.Errorf("unexpected http status code: %v", r.StatusCode)
	}
	ctx := context.New(config.Project{ProjectName: "blah"})
	ctx.Env["TEST_A_SECRET"] = "x"
	ctx.Env["TEST_A_USERNAME"] = "u2"
	ctx.Version = "2.1.0"
	ctx.Artifacts = artifact.New()
	folder, err := ioutil.TempDir("", "goreleasertest")
	require.NoError(t, err)
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
		var file = filepath.Join(folder, "a."+a.ext)
		require.NoError(t, ioutil.WriteFile(file, []byte("lorem ipsum"), 0644))
		ctx.Artifacts.Add(artifact.Artifact{Name: "a." + a.ext, Path: file, Type: a.typ})
	}

	tests := []struct {
		name         string
		tryPlain     bool
		tryTLS       bool
		wantErrPlain bool
		wantErrTLS   bool
		setup        func(*httptest.Server) (*context.Context, config.Put)
		check        func(r []*h.Request) error
	}{
		{"wrong-mode", true, true, true, true,
			func(s *httptest.Server) (*context.Context, config.Put) {
				return ctx, config.Put{
					Mode:         "wrong-mode",
					Name:         "a",
					Target:       s.URL + "/{{.ProjectName}}/{{.Version}}/",
					Username:     "u1",
					TrustedCerts: cert(s),
				}
			},
			checks(),
		},
		{"username-from-env", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Put) {
				return ctx, config.Put{
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
		{"archive", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Put) {
				return ctx, config.Put{
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
		{"binary", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Put) {
				return ctx, config.Put{
					Mode:         ModeBinary,
					Name:         "a",
					Target:       s.URL + "/{{.ProjectName}}/{{.Version}}/",
					Username:     "u2",
					TrustedCerts: cert(s),
				}
			},
			checks(check{"/blah/2.1.0/a.ubi", "u2", "x", content, map[string]string{}}),
		},
		{"binary-add-ending-bar", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Put) {
				return ctx, config.Put{
					Mode:         ModeBinary,
					Name:         "a",
					Target:       s.URL + "/{{.ProjectName}}/{{.Version}}",
					Username:     "u2",
					TrustedCerts: cert(s),
				}
			},
			checks(check{"/blah/2.1.0/a.ubi", "u2", "x", content, map[string]string{}}),
		},
		{"archive-with-checksum-and-signature", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Put) {
				return ctx, config.Put{
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
		{"bad-template", true, true, true, true,
			func(s *httptest.Server) (*context.Context, config.Put) {
				return ctx, config.Put{
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
		{"failed-request", true, true, true, true,
			func(s *httptest.Server) (*context.Context, config.Put) {
				return ctx, config.Put{
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
		{"broken-cert", false, true, false, true,
			func(s *httptest.Server) (*context.Context, config.Put) {
				return ctx, config.Put{
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
		{"skip-publishing", true, true, true, true,
			func(s *httptest.Server) (*context.Context, config.Put) {
				c := *ctx
				c.SkipPublish = true
				return &c, config.Put{}
			},
			checks(),
		},
		{"checksumheader", true, true, false, false,
			func(s *httptest.Server) (*context.Context, config.Put) {
				return ctx, config.Put{
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
	}

	uploadAndCheck := func(setup func(*httptest.Server) (*context.Context, config.Put), wantErrPlain, wantErrTLS bool, check func(r []*h.Request) error, srv *httptest.Server) {
		requests = nil
		ctx, put := setup(srv)
		wantErr := wantErrPlain
		if srv.Certificate() != nil {
			wantErr = wantErrTLS
		}
		if err := Upload(ctx, []config.Put{put}, "test", is2xx); (err != nil) != wantErr {
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
				uploadAndCheck(tt.setup, tt.wantErrPlain, tt.wantErrTLS, tt.check, srv)
			})
		}
		if tt.tryTLS {
			t.Run(tt.name+"-tls", func(t *testing.T) {
				srv := httptest.NewUnstartedServer(mux)
				srv.StartTLS()
				defer srv.Close()
				uploadAndCheck(tt.setup, tt.wantErrPlain, tt.wantErrTLS, tt.check, srv)
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
