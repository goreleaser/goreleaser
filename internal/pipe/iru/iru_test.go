package iru

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestContinueOnError(t *testing.T) {
	require.True(t, Pipe{}.ContinueOnError())
}

func TestSkip(t *testing.T) {
	t.Run("skip flag", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Iru: config.Iru{URL: "https://acme.api.kandji.io"},
		})
		skips.Set(ctx, skips.Iru)
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("skip no url", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Iru: config.Iru{URL: "https://acme.api.kandji.io"},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "myapp",
		Iru:         config.Iru{URL: "https://acme.api.kandji.io"},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "myapp", ctx.Config.Iru.Name)
	require.Equal(t, "package", ctx.Config.Iru.InstallType)
	require.Equal(t, "install_once", ctx.Config.Iru.InstallEnforcement)
}

func TestDefaultKeepsUserValues(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "myapp",
		Iru: config.Iru{
			Name:               "Custom Name",
			APIToken:           "token",
			InstallType:        "zip",
			InstallEnforcement: "no_enforcement",
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "Custom Name", ctx.Config.Iru.Name)
	require.Equal(t, "token", ctx.Config.Iru.APIToken)
	require.Equal(t, "zip", ctx.Config.Iru.InstallType)
	require.Equal(t, "no_enforcement", ctx.Config.Iru.InstallEnforcement)
}

func TestPublishDisabled(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:     "https://acme.api.kandji.io",
			Name:    "myapp",
			Disable: "true",
		},
	})
	testlib.AssertSkipped(t, Pipe{}.Publish(ctx))
}

func TestPublishMissingToken(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:  "https://acme.api.kandji.io",
			Name: "myapp",
		},
	})
	require.ErrorContains(t, Pipe{}.Publish(ctx), "missing API token")
}

func TestPublishTokenFromEnv(t *testing.T) {
	srv := newTestServer(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "myapp",
		Iru: config.Iru{
			URL:                srv.URL,
			Name:               "My App",
			InstallType:        "package",
			InstallEnforcement: "install_once",
		},
	}, testctx.WithEnv(map[string]string{"IRU_API_TOKEN": "token"}))
	ctx.Artifacts.Add(testArtifact(t))

	require.NoError(t, Pipe{}.Publish(ctx))
	require.Equal(t, 1, srv.uploadInits)
}

func TestPublishNoArtifacts(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:      "https://acme.api.kandji.io",
			Name:     "myapp",
			APIToken: "token",
		},
	})
	testlib.AssertSkipped(t, Pipe{}.Publish(ctx))
}

func TestPublishUpdateWithMultipleArtifacts(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:           "https://acme.api.kandji.io",
			Name:          "myapp",
			APIToken:      "token",
			LibraryItemID: "some-id",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "a.pkg", Path: "a.pkg", Type: artifact.UploadableFile,
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "b.pkg", Path: "b.pkg", Type: artifact.UploadableFile,
	})
	require.ErrorContains(t, Pipe{}.Publish(ctx), "library_item_id is set")
}

type testServer struct {
	*httptest.Server

	// mu guards the fields below: the pipe publishes artifacts in
	// parallel, so handlers run concurrently.
	mu             sync.Mutex
	uploadInits    int
	s3Uploads      int
	s3Fields       map[string]string
	s3FileContent  string
	saveCalls      int
	saveMethod     string
	savePath       string
	saveForm       map[string]string
	failInitStatus int
	failS3Status   int
	failSaveStatus int
	failSaveTimes  int
	emptyUploadRes bool
}

func newTestServer(tb testing.TB) *testServer {
	tb.Helper()
	ts := &testServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/library/custom-apps/upload", func(w http.ResponseWriter, r *http.Request) {
		ts.mu.Lock()
		defer ts.mu.Unlock()
		ts.uploadInits++
		if r.Header.Get("Authorization") != "Bearer token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if ts.failInitStatus != 0 {
			w.WriteHeader(ts.failInitStatus)
			return
		}
		var body struct {
			Name string `json:"name"`
		}
		assert.NoError(tb, json.NewDecoder(r.Body).Decode(&body))
		assert.NotEmpty(tb, body.Name)
		if ts.emptyUploadRes {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("{}"))
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"post_url": ts.URL + "/s3-upload",
			"post_data": map[string]string{
				"key":    "companies/xyz/" + body.Name,
				"policy": "some-policy",
			},
			"file_key": "companies/xyz/" + body.Name,
		})
	})
	mux.HandleFunc("POST /s3-upload", func(w http.ResponseWriter, r *http.Request) {
		ts.mu.Lock()
		defer ts.mu.Unlock()
		ts.s3Uploads++
		// S3 rejects requests without a Content-Length with a 411.
		if r.ContentLength <= 0 {
			w.WriteHeader(http.StatusLengthRequired)
			return
		}
		if ts.failS3Status != 0 {
			w.WriteHeader(ts.failS3Status)
			return
		}
		if !assert.NoError(tb, r.ParseMultipartForm(1<<20)) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ts.s3Fields = map[string]string{}
		for k, v := range r.MultipartForm.Value {
			ts.s3Fields[k] = v[0]
		}
		file, _, err := r.FormFile("file")
		if !assert.NoError(tb, err) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		content, err := io.ReadAll(file)
		assert.NoError(tb, err)
		ts.s3FileContent = string(content)
		w.WriteHeader(http.StatusNoContent)
	})
	saveHandler := func(w http.ResponseWriter, r *http.Request) {
		ts.mu.Lock()
		defer ts.mu.Unlock()
		ts.saveCalls++
		if r.Header.Get("Authorization") != "Bearer token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if ts.failSaveStatus != 0 {
			w.WriteHeader(ts.failSaveStatus)
			return
		}
		// Simulates the API rejecting the create right after the S3 upload
		// with "The upload is still being processed".
		if ts.failSaveTimes > 0 {
			ts.failSaveTimes--
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"detail":"The upload is still being processed."}`))
			return
		}
		ts.saveMethod = r.Method
		ts.savePath = r.URL.Path
		assert.NoError(tb, r.ParseForm())
		ts.saveForm = map[string]string{}
		for k, v := range r.PostForm {
			ts.saveForm[k] = v[0]
		}
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id":   "58429143-b55c-42d3-a9a3-7c699ddd0ce1",
			"name": ts.saveForm["name"],
		})
	}
	mux.HandleFunc("POST /api/v1/library/custom-apps", saveHandler)
	mux.HandleFunc("PATCH /api/v1/library/custom-apps/{id}", saveHandler)
	ts.Server = httptest.NewServer(mux)
	tb.Cleanup(ts.Close)
	return ts
}

const testFileContent = "fake pkg content"

func testArtifact(tb testing.TB) *artifact.Artifact {
	tb.Helper()
	// The on-disk file name intentionally differs from the artifact name to
	// ensure uploads are named after the artifact, not its path.
	path := filepath.Join(tb.TempDir(), "binary")
	require.NoError(tb, os.WriteFile(path, []byte(testFileContent), 0o644))
	return &artifact.Artifact{
		Name: "myapp.pkg",
		Path: path,
		Type: artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID: "default",
		},
	}
}

func TestPublish(t *testing.T) {
	srv := newTestServer(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "myapp",
		Iru: config.Iru{
			URL:                   srv.URL,
			Name:                  "My App {{ .Version }}",
			APIToken:              "token",
			InstallType:           "package",
			InstallEnforcement:    "install_once",
			ShowInSelfService:     new(true),
			SelfServiceCategoryID: "cat-id",
		},
	}, testctx.WithVersion("1.2.3"))
	ctx.Artifacts.Add(testArtifact(t))

	require.NoError(t, Pipe{}.Publish(ctx))

	require.Equal(t, 1, srv.uploadInits)
	require.Equal(t, 1, srv.s3Uploads)
	require.Equal(t, "companies/xyz/myapp.pkg", srv.s3Fields["key"])
	require.Equal(t, "some-policy", srv.s3Fields["policy"])
	require.Equal(t, testFileContent, srv.s3FileContent)

	require.Equal(t, http.MethodPost, srv.saveMethod)
	require.Equal(t, "/api/v1/library/custom-apps", srv.savePath)
	require.Equal(t, "My App 1.2.3", srv.saveForm["name"])
	require.Equal(t, "companies/xyz/myapp.pkg", srv.saveForm["file_key"])
	require.Equal(t, "package", srv.saveForm["install_type"])
	require.Equal(t, "install_once", srv.saveForm["install_enforcement"])
	require.Equal(t, "true", srv.saveForm["show_in_self_service"])
	require.Equal(t, "cat-id", srv.saveForm["self_service_category_id"])
	require.NotContains(t, srv.saveForm, "restart")
	require.NotContains(t, srv.saveForm, "self_service_recommended")
	require.NotContains(t, srv.saveForm, "unzip_location")
	require.NotContains(t, srv.saveForm, "preinstall_script")
}

func TestPublishUpdate(t *testing.T) {
	srv := newTestServer(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "myapp",
		Iru: config.Iru{
			URL:                srv.URL,
			Name:               "My App",
			APIToken:           "token",
			LibraryItemID:      "some-lib-id",
			InstallType:        "package",
			InstallEnforcement: "install_once",
		},
	})
	ctx.Artifacts.Add(testArtifact(t))

	require.NoError(t, Pipe{}.Publish(ctx))

	require.Equal(t, http.MethodPatch, srv.saveMethod)
	require.Equal(t, "/api/v1/library/custom-apps/some-lib-id", srv.savePath)
	require.Equal(t, "companies/xyz/myapp.pkg", srv.saveForm["file_key"])
}

func TestPublishInitUploadError(t *testing.T) {
	srv := newTestServer(t)
	srv.failInitStatus = http.StatusBadRequest
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:      srv.URL,
			Name:     "My App",
			APIToken: "token",
		},
	})
	ctx.Artifacts.Add(testArtifact(t))

	require.ErrorContains(t, Pipe{}.Publish(ctx), "could not initialize upload")
	require.Equal(t, 0, srv.s3Uploads)
}

func TestPublishValidation(t *testing.T) {
	for name, cfg := range map[string]config.Iru{
		"zip without unzip_location": {
			InstallType: "zip",
		},
		"continuously_enforce without audit_script": {
			InstallEnforcement: "continuously_enforce",
		},
		"self service without category": {
			ShowInSelfService: new(true),
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg.URL = "https://acme.api.kandji.io"
			cfg.APIToken = "token"
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{Iru: cfg})
			require.ErrorContains(t, Pipe{}.Publish(ctx), "is not set")
		})
	}
}

func TestPublishEmptyTemplatedURL(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:      "{{ .Env.IRU_URL }}",
			APIToken: "token",
		},
	}, testctx.WithEnv(map[string]string{"IRU_URL": ""}))
	require.ErrorContains(t, Pipe{}.Publish(ctx), "url templated to an empty string")
}

func TestPublishCreateRetriesWhileProcessing(t *testing.T) {
	srv := newTestServer(t)
	srv.failSaveTimes = 2
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:      srv.URL,
			Name:     "My App",
			APIToken: "token",
		},
		Retry: config.Retry{
			Attempts: 5,
			Delay:    time.Millisecond,
			MaxDelay: time.Millisecond,
		},
	})
	ctx.Artifacts.Add(testArtifact(t))

	require.NoError(t, Pipe{}.Publish(ctx))
	require.Equal(t, http.MethodPost, srv.saveMethod)
	require.Equal(t, 3, srv.saveCalls)
}

func TestPublishCreateDoesNotRetryAmbiguousErrors(t *testing.T) {
	srv := newTestServer(t)
	srv.failSaveStatus = http.StatusBadGateway
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:      srv.URL,
			Name:     "My App",
			APIToken: "token",
		},
		Retry: config.Retry{
			Attempts: 5,
			Delay:    time.Millisecond,
			MaxDelay: time.Millisecond,
		},
	})
	ctx.Artifacts.Add(testArtifact(t))

	require.ErrorContains(t, Pipe{}.Publish(ctx), "could not save custom app")
	require.Equal(t, 1, srv.saveCalls)
}

func TestPublishMultipleArtifacts(t *testing.T) {
	srv := newTestServer(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:      srv.URL,
			Name:     "My App {{ .ArtifactName }}",
			APIToken: "token",
		},
	})
	first := testArtifact(t)
	second := testArtifact(t)
	second.Name = "other.pkg"
	ctx.Artifacts.Add(first)
	ctx.Artifacts.Add(second)

	require.NoError(t, Pipe{}.Publish(ctx))
	require.Equal(t, 2, srv.uploadInits)
	require.Equal(t, 2, srv.s3Uploads)
}

func TestPublishDisableTemplateError(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:     "https://acme.api.kandji.io",
			Disable: "{{ .Nope }}",
		},
	})
	require.ErrorContains(t, Pipe{}.Publish(ctx), "could not evaluate iru.disable")
}

func TestPublishURLTemplateError(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL: "{{ .Nope }}",
		},
	})
	require.ErrorContains(t, Pipe{}.Publish(ctx), "could not apply templates")
}

func TestPublishNameTemplateError(t *testing.T) {
	srv := newTestServer(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:      srv.URL,
			Name:     "{{ .Nope }}",
			APIToken: "token",
		},
	})
	ctx.Artifacts.Add(testArtifact(t))
	require.ErrorContains(t, Pipe{}.Publish(ctx), "could not apply templates to iru.name")
}

func TestPublishInvalidUploadResponse(t *testing.T) {
	srv := newTestServer(t)
	srv.emptyUploadRes = true
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:      srv.URL,
			Name:     "My App",
			APIToken: "token",
		},
	})
	ctx.Artifacts.Add(testArtifact(t))
	require.ErrorContains(t, Pipe{}.Publish(ctx), "missing post_url or file_key")
}

func TestPublishS3UploadError(t *testing.T) {
	srv := newTestServer(t)
	srv.failS3Status = http.StatusForbidden
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:      srv.URL,
			Name:     "My App",
			APIToken: "token",
		},
	})
	ctx.Artifacts.Add(testArtifact(t))
	require.ErrorContains(t, Pipe{}.Publish(ctx), "could not upload file")
	require.Empty(t, srv.saveMethod)
}

func TestPublishCreateError(t *testing.T) {
	srv := newTestServer(t)
	srv.failSaveStatus = http.StatusForbidden
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:      srv.URL,
			Name:     "My App",
			APIToken: "token",
		},
	})
	ctx.Artifacts.Add(testArtifact(t))
	require.ErrorContains(t, Pipe{}.Publish(ctx), "could not save custom app")
}

func TestPublishUpdateError(t *testing.T) {
	srv := newTestServer(t)
	srv.failSaveStatus = http.StatusNotFound
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:           srv.URL,
			Name:          "My App",
			APIToken:      "token",
			LibraryItemID: "some-lib-id",
		},
	})
	ctx.Artifacts.Add(testArtifact(t))
	require.ErrorContains(t, Pipe{}.Publish(ctx), "could not save custom app")
}

func TestPublishMissingFile(t *testing.T) {
	srv := newTestServer(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:      srv.URL,
			Name:     "My App",
			APIToken: "token",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "gone.pkg",
		Path: filepath.Join(t.TempDir(), "does-not-exist"),
		Type: artifact.UploadableArchive,
	})
	require.ErrorContains(t, Pipe{}.Publish(ctx), "could not upload file")
}

func TestPublishServerUnreachable(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:      "http://127.0.0.1:1",
			Name:     "My App",
			APIToken: "token",
		},
	})
	ctx.Artifacts.Add(testArtifact(t))
	require.ErrorContains(t, Pipe{}.Publish(ctx), "could not initialize upload")
}

func TestPublishFilterByIDs(t *testing.T) {
	srv := newTestServer(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "myapp",
		Iru: config.Iru{
			URL:                srv.URL,
			Name:               "My App",
			APIToken:           "token",
			IDs:                []string{"other"},
			InstallType:        "package",
			InstallEnforcement: "install_once",
		},
	})
	art := testArtifact(t)
	art.Type = artifact.UploadableArchive
	ctx.Artifacts.Add(art)

	// artifact has ID "default", filter wants "other": nothing matches.
	testlib.AssertSkipped(t, Pipe{}.Publish(ctx))
	require.Equal(t, 0, srv.uploadInits)
}
