package http

import (
	stdctx "context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestUploadTLSClientSetupError(t *testing.T) {
	ctx := uploadClientTestContext(t)
	upload := config.Upload{
		Name: "broken-client", Mode: ModeArchive, Method: http.MethodPut,
		Target:         "https://127.0.0.1:1",
		ClientX509Cert: filepath.Join(t.TempDir(), "missing.cert"), ClientX509Key: "missing.key",
	}
	// Certificate files can become unavailable after CheckConfig succeeds.
	err := uploadWithFilter(ctx, &upload, artifact.ByTypes(artifact.UploadableArchive), "test", uploadClientTestCheck)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.ErrorContains(t, err, "broken-client: test: upload failed:")
}

func TestUploadReusesTLSClient(t *testing.T) {
	for _, retry := range []bool{false, true} {
		t.Run(fmt.Sprintf("retry=%t", retry), func(t *testing.T) {
			ctx := uploadClientTestContext(t)
			ctx.Parallelism = 1
			var requests []string
			attempts := map[string]int{}
			var requestMu sync.Mutex
			connections := newUploadConnections()
			srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				username, password, _ := r.BasicAuth()
				requestMu.Lock()
				requests = append(requests, fmt.Sprintf("%s %s %s %s %s %s", r.Method, r.URL.Path, body, username, password, r.Header.Get("X-Artifact")))
				attempts[r.URL.Path]++
				attempt := attempts[r.URL.Path]
				requestMu.Unlock()
				// An empty response allows reuse without changing response-body handling.
				w.Header().Set("Content-Length", "0")
				if retry && attempt == 1 {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				w.WriteHeader(http.StatusCreated)
			}))
			srv.Config.ConnState = connections.record
			srv.StartTLS()
			t.Cleanup(srv.Close)
			upload := uploadClientTestConfig(srv)
			upload.Username, upload.Password = "alice", "secret"
			upload.CustomHeaders = map[string]string{"X-Artifact": "{{ .ArtifactName }}"}
			require.NoError(t, Upload(ctx, []config.Upload{upload}, "test", uploadClientTestCheck))
			want := []string{}
			for i := range 3 {
				request := fmt.Sprintf("PUT /asset-%d payload-%d alice secret asset-%d", i, i, i)
				want = append(want, request)
				if retry {
					want = append(want, request)
				}
			}
			requestMu.Lock()
			got := append([]string(nil), requests...)
			requestMu.Unlock()
			require.Equal(t, want, got)
			created, _, _ := connections.counts()
			require.Equal(t, 1, created, "all artifacts and attempts should share a connection")
			connections.requireClosed(t)
		})
	}
}

func TestUploadTLSClientCleanupWaitsForWorkers(t *testing.T) {
	for _, cancelUpload := range []bool{false, true} {
		t.Run(fmt.Sprintf("cancel=%t", cancelUpload), func(t *testing.T) {
			ctx := uploadClientTestContext(t)
			ctx.Parallelism = 3
			ctx.Config.Retry.Attempts = 1
			parent, cancel := stdctx.WithCancel(ctx.Context)
			ctx.Context = parent
			defer cancel()
			release := make(chan struct{})
			var releaseOnce sync.Once
			unblock := func() { releaseOnce.Do(func() { close(release) }) }
			allStarted := make(chan struct{})
			var started atomic.Int32
			connections := newUploadConnections()
			srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				if started.Add(1) == 3 {
					close(allStarted)
				}
				select {
				case <-allStarted:
				case <-r.Context().Done():
					return
				}
				if r.URL.Path == "/asset-2" {
					select {
					case <-release:
					case <-r.Context().Done():
						return
					}
				}
				w.Header().Set("Content-Length", "0")
				if !cancelUpload && r.URL.Path == "/asset-0" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusCreated)
			}))
			srv.Config.ConnState = connections.record
			srv.StartTLS()
			t.Cleanup(srv.Close)
			t.Cleanup(unblock)
			done := make(chan error, 1)
			go func() {
				done <- Upload(ctx, []config.Upload{uploadClientTestConfig(srv)}, "test", uploadClientTestCheck)
			}()
			require.Eventually(t, func() bool {
				created, open, idle := connections.counts()
				return created == 3 && open == 3 && idle == 2
			}, 5*time.Second, time.Millisecond, "idle connections must stay open while another worker is active")
			select {
			case err := <-done:
				t.Fatalf("upload returned before the last worker: %v", err)
			default:
			}
			if cancelUpload {
				cancel()
			} else {
				unblock()
			}
			select {
			case err := <-done:
				if cancelUpload {
					require.ErrorIs(t, err, stdctx.Canceled)
				} else {
					require.ErrorContains(t, err, "unexpected status: 400")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("upload did not finish")
			}
			connections.requireClosed(t)
		})
	}
}

func TestUploadBorrowedDefaultClient(t *testing.T) {
	ctx := uploadClientTestContext(t)
	ctx.Parallelism = 1
	connections := newUploadConnections()
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusCreated)
	}))
	srv.Config.ConnState = connections.record
	srv.StartTLS()
	t.Cleanup(srv.Close)
	borrowed := srv.Client()
	transport := &uploadCloseTracker{RoundTripper: borrowed.Transport}
	borrowed.Transport = transport
	previous := http.DefaultClient
	http.DefaultClient = borrowed
	t.Cleanup(func() {
		http.DefaultClient = previous
		borrowed.CloseIdleConnections()
	})
	upload := uploadClientTestConfig(srv)
	upload.TrustedCerts = ""
	require.NoError(t, Upload(ctx, []config.Upload{upload}, "test", uploadClientTestCheck))
	require.Zero(t, transport.closed.Load(), "the uploader does not own the default client")
	created, open, _ := connections.counts()
	require.Equal(t, 1, created)
	require.Equal(t, 1, open)
	borrowed.CloseIdleConnections()
	connections.requireClosed(t)
}

func TestUploadTLSClientConfigurationIsolation(t *testing.T) {
	ctx := uploadClientTestContext(t)
	ctx.Parallelism = 1
	pool := x509.NewCertPool()
	configurations := make([]config.Upload, 2)
	for i, name := range []string{"alice", "bob"} {
		certFile, keyFile, certificate := uploadClientCertificate(t, name)
		pool.AddCert(certificate)
		configurations[i] = config.Upload{
			Name: name, Mode: ModeArchive, Method: http.MethodPut,
			Username: name, Password: name + "-secret",
			ClientX509Cert: certFile, ClientX509Key: keyFile,
		}
	}
	var requests atomic.Int32
	connections := newUploadConnections()
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		username, password, _ := r.BasicAuth()
		name, _, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
		if username != name || password != name+"-secret" || len(r.TLS.PeerCertificates) != 1 || r.TLS.PeerCertificates[0].Subject.CommonName != name {
			t.Errorf("incorrect credentials or client certificate for %s", name)
		}
		requests.Add(1)
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusCreated)
	}))
	srv.TLS = &tls.Config{ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: pool}
	srv.Config.ConnState = connections.record
	srv.StartTLS()
	t.Cleanup(srv.Close)
	for i := range configurations {
		configurations[i].Target = srv.URL + "/" + configurations[i].Name
		configurations[i].TrustedCerts = cert(srv)
	}
	require.NoError(t, Upload(ctx, configurations, "test", uploadClientTestCheck))
	require.EqualValues(t, 6, requests.Load())
	created, _, _ := connections.counts()
	require.Equal(t, 2, created, "each configuration should own one separate connection")
	connections.requireClosed(t)
}

func uploadClientTestContext(t *testing.T) *context.Context {
	t.Helper()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "test",
		Retry:       config.Retry{Attempts: 2, Delay: time.Millisecond, MaxDelay: time.Millisecond},
	})
	dir := t.TempDir()
	for i := range 3 {
		name := fmt.Sprintf("asset-%d", i)
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, fmt.Appendf(nil, "payload-%d", i), 0o600))
		ctx.Artifacts.Add(&artifact.Artifact{Name: name, Path: path, Type: artifact.UploadableArchive})
	}
	return ctx
}

func uploadClientTestConfig(srv *httptest.Server) config.Upload {
	return config.Upload{Name: "test", Mode: ModeArchive, Method: http.MethodPut, Target: srv.URL, TrustedCerts: cert(srv)}
}

func uploadClientTestCheck(resp *http.Response) error {
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}

type uploadConnections struct {
	mu      sync.Mutex
	states  map[net.Conn]http.ConnState
	created int
}

func newUploadConnections() *uploadConnections {
	return &uploadConnections{states: map[net.Conn]http.ConnState{}}
}

func (c *uploadConnections) record(conn net.Conn, state http.ConnState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if state == http.StateNew {
		c.created++
	}
	if state == http.StateClosed {
		delete(c.states, conn)
	} else {
		c.states[conn] = state
	}
}

func (c *uploadConnections) counts() (created, open, idle int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, state := range c.states {
		if state == http.StateIdle {
			idle++
		}
	}
	return c.created, len(c.states), idle
}

func (c *uploadConnections) requireClosed(t *testing.T) {
	t.Helper()
	require.Eventually(t, func() bool {
		_, open, _ := c.counts()
		return open == 0
	}, 5*time.Second, time.Millisecond, "owned idle connections should close after upload")
}

type uploadCloseTracker struct {
	http.RoundTripper
	closed atomic.Int32
}

func (t *uploadCloseTracker) CloseIdleConnections() {
	t.closed.Add(1)
	t.RoundTripper.(*http.Transport).CloseIdleConnections()
}

func uploadClientCertificate(t *testing.T, name string) (certFile, keyFile string, certificate *x509.Certificate) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: name},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, privateKey)
	require.NoError(t, err)
	certificate, err = x509.ParseCertificate(der)
	require.NoError(t, err)
	keyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)
	dir := t.TempDir()
	certFile, keyFile = filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0o600))
	return certFile, keyFile, certificate
}
