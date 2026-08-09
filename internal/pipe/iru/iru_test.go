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
			Iru: config.Iru{URL: "https://iru.invalid"},
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
			Iru: config.Iru{URL: "https://iru.invalid"},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestPublishDisabled(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:     "https://iru.invalid",
			Name:    "myapp",
			Disable: "true",
		},
	})
	testlib.AssertSkipped(t, Pipe{}.Publish(ctx))
}

func TestPublishMissingToken(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:  "https://iru.invalid",
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
			URL:      "https://iru.invalid",
			Name:     "myapp",
			APIToken: "token",
		},
	})
	testlib.AssertSkipped(t, Pipe{}.Publish(ctx))
}

func TestPublishUpdateWithMultipleArtifacts(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:           "https://iru.invalid",
			Name:          "myapp",
			APIToken:      "token",
			LibraryItemID: "some-id",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "a.pkg", Path: "a.pkg", Goos: "darwin", Type: artifact.UploadableArchive,
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "b.pkg", Path: "b.pkg", Goos: "darwin", Type: artifact.UploadableArchive,
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
			_, _ = io.WriteString(w, "{}")
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
			_, _ = io.WriteString(w, `{"detail":"The upload is still being processed."}`)
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
		Goos: "darwin",
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
			SelfServiceCategoryID: new("cat-id"),
			Active:                new(false),
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
	require.Equal(t, "false", srv.saveForm["active"])
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
			URL:           srv.URL,
			APIToken:      "token",
			LibraryItemID: "some-lib-id",
		},
	})
	ctx.Artifacts.Add(testArtifact(t))

	require.NoError(t, Pipe{}.Publish(ctx))

	require.Equal(t, http.MethodPatch, srv.saveMethod)
	require.Equal(t, "/api/v1/library/custom-apps/some-lib-id", srv.savePath)
	require.Equal(t, "companies/xyz/myapp.pkg", srv.saveForm["file_key"])
	// Updates only send explicitly configured fields, so settings managed
	// in the Iru dashboard are kept.
	require.NotContains(t, srv.saveForm, "name")
	require.NotContains(t, srv.saveForm, "install_type")
	require.NotContains(t, srv.saveForm, "install_enforcement")
	require.NotContains(t, srv.saveForm, "active")
}

func TestPublishUpdateSendsExplicitFields(t *testing.T) {
	srv := newTestServer(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "myapp",
		Iru: config.Iru{
			URL:           srv.URL,
			Name:          "My App",
			APIToken:      "token",
			LibraryItemID: "some-lib-id",
			InstallType:   "package",
		},
	})
	ctx.Artifacts.Add(testArtifact(t))

	require.NoError(t, Pipe{}.Publish(ctx))

	require.Equal(t, http.MethodPatch, srv.saveMethod)
	require.Equal(t, "My App", srv.saveForm["name"])
	require.Equal(t, "package", srv.saveForm["install_type"])
	require.NotContains(t, srv.saveForm, "install_enforcement")
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

// validationCase is a config that must be rejected, along with the message of
// the rule that has to reject it: asserting the message keeps these tables from
// passing for another reason, e.g. because no artifact matched.
type validationCase struct {
	cfg     config.Iru
	wantErr string
}

func TestPublishValidation(t *testing.T) {
	for name, tt := range map[string]validationCase{
		"zip without unzip_location": {
			cfg:     config.Iru{InstallType: "zip"},
			wantErr: "install_type is zip, but unzip_location is not set",
		},
		"unzip_location without zip": {
			cfg:     config.Iru{InstallType: "package", UnzipLocation: new("/Applications")},
			wantErr: "unzip_location is set, but install_type is package instead of zip",
		},
		"continuously_enforce without audit_script": {
			cfg:     config.Iru{InstallEnforcement: "continuously_enforce"},
			wantErr: "install_enforcement is continuously_enforce, but audit_script is not set",
		},
		"audit_script without continuously_enforce": {
			cfg:     config.Iru{AuditScript: new("#!/bin/zsh")},
			wantErr: "audit_script is set, but install_enforcement is install_once instead of continuously_enforce",
		},
		"no_enforcement without self service": {
			cfg:     config.Iru{InstallEnforcement: "no_enforcement"},
			wantErr: "install_enforcement is no_enforcement, but show_in_self_service is not enabled",
		},
		"self service without category": {
			cfg:     config.Iru{ShowInSelfService: new(true)},
			wantErr: "show_in_self_service is enabled, but self_service_category_id is not set",
		},
		"category without self service": {
			cfg:     config.Iru{SelfServiceCategoryID: new("cat-id")},
			wantErr: "self_service_category_id is set, but show_in_self_service is not enabled",
		},
		"recommended without self service": {
			cfg:     config.Iru{SelfServiceRecommended: new(true)},
			wantErr: "self_service_recommended is enabled, but show_in_self_service is not enabled",
		},
		"invalid install type": {
			cfg:     config.Iru{InstallType: "nope"},
			wantErr: "invalid install_type: nope",
		},
		"invalid install enforcement": {
			cfg:     config.Iru{InstallEnforcement: "once"},
			wantErr: "invalid install_enforcement: once",
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := tt.cfg
			cfg.URL = "https://iru.invalid"
			cfg.APIToken = "token"
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{Iru: cfg})
			ctx.Artifacts.Add(testArtifact(t))

			require.ErrorContains(t, Pipe{}.Publish(ctx), tt.wantErr)
		})
	}
}

func TestPublishValidationOnUpdate(t *testing.T) {
	t.Run("companion fields may live on the remote item", func(t *testing.T) {
		srv := newTestServer(t)
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Iru: config.Iru{
				URL:      srv.URL,
				APIToken: "token",
				// unzip_location and the Self Service category are not set
				// here: the existing library item already has them.
				LibraryItemID:     "some-lib-id",
				InstallType:       "zip",
				ShowInSelfService: new(true),
			},
		})
		art := testArtifact(t)
		art.Name = "myapp.zip"
		ctx.Artifacts.Add(art)

		require.NoError(t, Pipe{}.Publish(ctx))
	})

	// Conflicts the pipe can see without knowing the remote item must still
	// fail before the upload, even when updating.
	for name, tt := range map[string]validationCase{
		"zip with cleared unzip_location": {
			cfg:     config.Iru{InstallType: "zip", UnzipLocation: new("")},
			wantErr: "install_type is zip, but unzip_location is not set",
		},
		"unzip_location with image": {
			cfg:     config.Iru{InstallType: "image", UnzipLocation: new("/Applications")},
			wantErr: "unzip_location is set, but install_type is image instead of zip",
		},
		"continuously_enforce with cleared audit_script": {
			cfg:     config.Iru{InstallEnforcement: "continuously_enforce", AuditScript: new("")},
			wantErr: "install_enforcement is continuously_enforce, but audit_script is not set",
		},
		"audit_script with install_once": {
			cfg:     config.Iru{InstallEnforcement: "install_once", AuditScript: new("#!/bin/zsh")},
			wantErr: "audit_script is set, but install_enforcement is install_once instead of continuously_enforce",
		},
		"no_enforcement with self service off": {
			cfg:     config.Iru{InstallEnforcement: "no_enforcement", ShowInSelfService: new(false)},
			wantErr: "install_enforcement is no_enforcement, but show_in_self_service is not enabled",
		},
		"self service with cleared category": {
			cfg:     config.Iru{ShowInSelfService: new(true), SelfServiceCategoryID: new("")},
			wantErr: "show_in_self_service is enabled, but self_service_category_id is not set",
		},
		"category with self service off": {
			cfg:     config.Iru{ShowInSelfService: new(false), SelfServiceCategoryID: new("cat-id")},
			wantErr: "self_service_category_id is set, but show_in_self_service is not enabled",
		},
		"recommended with self service off": {
			cfg:     config.Iru{ShowInSelfService: new(false), SelfServiceRecommended: new(true)},
			wantErr: "self_service_recommended is enabled, but show_in_self_service is not enabled",
		},
	} {
		t.Run("explicit conflict: "+name, func(t *testing.T) {
			server := newTestServer(t)
			cfg := tt.cfg
			cfg.URL = server.URL
			cfg.APIToken = "token"
			cfg.LibraryItemID = "some-lib-id"
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{Iru: cfg})
			ctx.Artifacts.Add(testArtifact(t))

			require.ErrorContains(t, Pipe{}.Publish(ctx), tt.wantErr)
			require.Equal(t, 0, server.uploadInits)
		})
	}
}

func TestPublishLibraryItemIDTemplatedEmpty(t *testing.T) {
	server := newTestServer(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:           server.URL,
			APIToken:      "token",
			LibraryItemID: "{{ .Env.ITEM_ID }}",
		},
	}, testctx.WithEnv(map[string]string{"ITEM_ID": ""}))
	ctx.Artifacts.Add(testArtifact(t))

	// Otherwise this would silently create a new Custom App.
	require.ErrorContains(t, Pipe{}.Publish(ctx), "library_item_id templated to an empty string")
	require.Equal(t, 0, server.uploadInits)
}

func TestPublishUpdateClearsExplicitlyEmptyFields(t *testing.T) {
	server := newTestServer(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:           server.URL,
			APIToken:      "token",
			LibraryItemID: "some-lib-id",
			AuditScript:   new(""),
		},
	})
	ctx.Artifacts.Add(testArtifact(t))

	require.NoError(t, Pipe{}.Publish(ctx))

	require.Equal(t, http.MethodPatch, server.saveMethod)
	// An explicitly empty value is sent, so it clears the field.
	require.Contains(t, server.saveForm, "audit_script")
	require.Empty(t, server.saveForm["audit_script"])
	require.NotContains(t, server.saveForm, "preinstall_script")
}

func TestPublishIgnoresNonMacOSArtifacts(t *testing.T) {
	server := newTestServer(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:           server.URL,
			APIToken:      "token",
			InstallType:   "zip",
			UnzipLocation: new("/usr/local/bin"),
		},
	})
	// A cross-platform release: only the darwin artifact is published, the
	// others are ignored instead of failing the release.
	mac := testArtifact(t)
	mac.Name = "myapp_darwin_all.zip"
	ctx.Artifacts.Add(mac)
	for _, goos := range []string{"windows", "linux"} {
		other := testArtifact(t)
		other.Name = "myapp_" + goos + "_amd64.zip"
		other.Goos = goos
		ctx.Artifacts.Add(other)
	}

	require.NoError(t, Pipe{}.Publish(ctx))
	require.Equal(t, 1, server.uploadInits)
	require.Equal(t, "companies/xyz/myapp_darwin_all.zip", server.saveForm["file_key"])
}

func TestPublishSkipsWhenOnlyNonMacOSArtifacts(t *testing.T) {
	server := newTestServer(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:      server.URL,
			APIToken: "token",
		},
	})
	art := testArtifact(t)
	art.Name = "myapp_windows_amd64.zip"
	art.Goos = "windows"
	ctx.Artifacts.Add(art)

	testlib.AssertSkipped(t, Pipe{}.Publish(ctx))
	require.Equal(t, 0, server.uploadInits)
}

func TestPublishArtifactUnsupportedExtOnUpdate(t *testing.T) {
	server := newTestServer(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:           server.URL,
			APIToken:      "token",
			LibraryItemID: "some-lib-id",
		},
	})
	// install_type is unset, but a tarball is never a Custom App file.
	art := testArtifact(t)
	art.Name = "myapp_darwin_all.tar.gz"
	ctx.Artifacts.Add(art)

	require.ErrorContains(t, Pipe{}.Publish(ctx), "is not a Custom App file")
	require.Equal(t, 0, server.uploadInits)
}

func TestPublishArtifactInstallTypeMismatch(t *testing.T) {
	srv := newTestServer(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Iru: config.Iru{
			URL:      srv.URL,
			APIToken: "token",
		},
	})
	// Default install_type is package, but the artifact is a tarball.
	art := testArtifact(t)
	art.Name = "myapp_linux_amd64.tar.gz"
	ctx.Artifacts.Add(art)

	require.ErrorContains(t, Pipe{}.Publish(ctx), "not compatible with install_type package")
	require.Equal(t, 0, srv.uploadInits)
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
			URL:     "https://iru.invalid",
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
		Goos: "darwin",
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
