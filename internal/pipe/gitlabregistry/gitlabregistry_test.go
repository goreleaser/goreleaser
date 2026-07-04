package gitlabregistry

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func requireMethodPut(t *testing.T, r *http.Request) {
	t.Helper()
	require.Equal(t, http.MethodPut, r.Method)
}

func requireHeader(t *testing.T, r *http.Request, header, want string) {
	t.Helper()
	require.Equal(t, want, r.Header.Get(header))
}

func TestRunPipe_ModeArchive(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	folder := t.TempDir()
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, tarfile.Close())
	debfile, err := os.Create(filepath.Join(folder, "bin.deb"))
	require.NoError(t, err)
	require.NoError(t, debfile.Close())

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "goreleaser",
		Dist:        folder,
		GitLabRegistries: []config.Upload{
			{
				Name:   "production",
				Mode:   "archive",
				Target: fmt.Sprintf("%s/api/v4/projects/1234/packages/generic/{{ .ProjectName }}/{{ .Version }}/", server.URL),
				CustomHeaders: map[string]string{
					"PRIVATE-TOKEN": "{{ .Env.GITLAB_TOKEN }}",
				},
			},
		},
		Archives: []config.Archive{{}},
		Env:      []string{"GITLAB_TOKEN=glpat-secret"},
	}, testctx.WithVersion("1.0.0"))

	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debfile.Name(),
	})

	var uploads sync.Map

	mux.HandleFunc("/api/v4/projects/1234/packages/generic/goreleaser/1.0.0/bin.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		requireMethodPut(t, r)
		requireHeader(t, r, "PRIVATE-TOKEN", "glpat-secret")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"message":"201 Created"}`)
		uploads.Store("targz", true)
	})
	mux.HandleFunc("/api/v4/projects/1234/packages/generic/goreleaser/1.0.0/bin.deb", func(w http.ResponseWriter, r *http.Request) {
		requireMethodPut(t, r)
		requireHeader(t, r, "PRIVATE-TOKEN", "glpat-secret")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"message":"201 Created"}`)
		uploads.Store("deb", true)
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Publish(ctx))
	_, ok := uploads.Load("targz")
	require.True(t, ok, "tar.gz file was not uploaded")
	_, ok = uploads.Load("deb")
	require.True(t, ok, "deb file was not uploaded")
}

func TestRunPipe_BasicAuth(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	folder := t.TempDir()
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, tarfile.Close())

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "goreleaser",
		Dist:        folder,
		GitLabRegistries: []config.Upload{
			{
				Name:     "production",
				Mode:     "archive",
				Target:   fmt.Sprintf("%s/api/v4/projects/mygroup%%2Freleases/packages/generic/{{ .ProjectName }}/{{ .Version }}/", server.URL),
				Username: "deploytoken",
			},
		},
		Archives: []config.Archive{{}},
		Env:      []string{"GITLAB_REGISTRY_PRODUCTION_SECRET=deploytoken-secret"},
	}, testctx.WithVersion("1.0.0"))

	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
	})

	var uploads sync.Map

	mux.HandleFunc("/api/v4/projects/mygroup%2Freleases/packages/generic/goreleaser/1.0.0/bin.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		requireMethodPut(t, r)
		// Basic auth of user "deploytoken" with secret "deploytoken-secret"
		requireHeader(t, r, "Authorization", "Basic ZGVwbG95dG9rZW46ZGVwbG95dG9rZW4tc2VjcmV0")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"message":"201 Created"}`)
		uploads.Store("targz", true)
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Publish(ctx))
	_, ok := uploads.Load("targz")
	require.True(t, ok, "tar.gz file was not uploaded")
}

func TestRunPipe_BadCredentials(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	folder := t.TempDir()
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, tarfile.Close())

	mux.HandleFunc("/api/v4/projects/1234/packages/generic/goreleaser/1.0.0/bin.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"message":"401 Unauthorized"}`)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "goreleaser",
		Dist:        folder,
		GitLabRegistries: []config.Upload{
			{
				Name:     "production",
				Mode:     "archive",
				Target:   fmt.Sprintf("%s/api/v4/projects/1234/packages/generic/{{ .ProjectName }}/{{ .Version }}/", server.URL),
				Username: "deployuser",
			},
		},
		Archives: []config.Archive{{}},
		Env:      []string{"GITLAB_REGISTRY_PRODUCTION_SECRET=deployuser-secret"},
	}, testctx.WithVersion("1.0.0"))

	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
	})

	require.NoError(t, Pipe{}.Default(ctx))
	err = Pipe{}.Publish(ctx)
	require.ErrorContains(t, err, "401 Unauthorized")
}

func TestRunPipe_ErrorField(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	folder := t.TempDir()
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, tarfile.Close())

	mux.HandleFunc("/api/v4/projects/1234/packages/generic/goreleaser/1.0.0/bin.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error":"insufficient_scope"}`)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "goreleaser",
		Dist:        folder,
		GitLabRegistries: []config.Upload{
			{
				Name:     "production",
				Mode:     "archive",
				Target:   fmt.Sprintf("%s/api/v4/projects/1234/packages/generic/{{ .ProjectName }}/{{ .Version }}/", server.URL),
				Username: "deployuser",
			},
		},
		Archives: []config.Archive{{}},
		Env:      []string{"GITLAB_REGISTRY_PRODUCTION_SECRET=deployuser-secret"},
	}, testctx.WithVersion("1.0.0"))

	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
	})

	require.NoError(t, Pipe{}.Default(ctx))
	err = Pipe{}.Publish(ctx)
	require.ErrorContains(t, err, "insufficient_scope")
}

func TestRunPipe_UnparsableErrorResponse(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	folder := t.TempDir()
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, tarfile.Close())

	mux.HandleFunc("/api/v4/projects/1234/packages/generic/goreleaser/1.0.0/bin.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `<body><h1>error</h1></body>`)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "goreleaser",
		Dist:        folder,
		GitLabRegistries: []config.Upload{
			{
				Name:     "production",
				Mode:     "archive",
				Target:   fmt.Sprintf("%s/api/v4/projects/1234/packages/generic/{{ .ProjectName }}/{{ .Version }}/", server.URL),
				Username: "deployuser",
			},
		},
		Archives: []config.Archive{{}},
		Env:      []string{"GITLAB_REGISTRY_PRODUCTION_SECRET=deployuser-secret"},
	}, testctx.WithVersion("1.0.0"))

	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Publish(ctx), `production: gitlab_registry: upload failed: unexpected error: invalid character '<' looking for beginning of value: <body><h1>error</h1></body>`)
}

func TestRunPipe_TargetTemplateError(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	binPath := filepath.Join(dist, "mybin", "mybin")

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		GitLabRegistries: []config.Upload{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   "http://gitlab.company.com/api/v4/projects/1234/packages/generic/{{.Name}",
				Username: "deployuser",
			},
		},
		Archives: []config.Archive{{}},
		Env:      []string{"GITLAB_REGISTRY_PRODUCTION_SECRET=deployuser-secret"},
	})

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	require.NoError(t, Pipe{}.Default(ctx))
	testlib.RequireTemplateError(t, Pipe{}.Publish(ctx))
}

func TestGitLabRegistriesWithoutTarget(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabRegistries: []config.Upload{
			{
				Name:     "production",
				Username: "deployuser",
			},
		},
		Env: []string{"GITLAB_REGISTRY_PRODUCTION_SECRET=deployuser-secret"},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	testlib.AssertSkipped(t, Pipe{}.Publish(ctx))
}

func TestGitLabRegistriesWithoutName(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabRegistries: []config.Upload{
			{
				Username: "deployuser",
				Target:   "http://gitlab.company.com/api/v4/projects/1234/packages/generic/{{ .ProjectName }}/{{ .Version }}/",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	testlib.AssertSkipped(t, Pipe{}.Publish(ctx))
}

func TestGitLabRegistriesWithoutSecret(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabRegistries: []config.Upload{
			{
				Name:     "production",
				Target:   "http://gitlab.company.com/api/v4/projects/1234/packages/generic/{{ .ProjectName }}/{{ .Version }}/",
				Username: "deployuser",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	testlib.AssertSkipped(t, Pipe{}.Publish(ctx))
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabRegistries: []config.Upload{
			{
				Name:     "production",
				Target:   "http://gitlab.company.com/api/v4/projects/1234/packages/generic/{{ .ProjectName }}/{{ .Version }}/",
				Username: "deployuser",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.GitLabRegistries, 1)
	registry := ctx.Config.GitLabRegistries[0]
	require.Equal(t, "archive", registry.Mode)
	require.Equal(t, http.MethodPut, registry.Method)
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitLabRegistries: []config.Upload{{}},
		})

		require.False(t, Pipe{}.Skip(ctx))
	})
}
