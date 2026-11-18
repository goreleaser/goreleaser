package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/goreleaser/goreleaser/v2/internal/pipe/release"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
)

func TestCustomProjectName(t *testing.T) {
	t.Parallel()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "foo",
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "bar",
				Name:  "bar",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "foo", ctx.Config.ProjectName)
}

func TestEmptyProjectName_DefaultsToGitHubRelease(t *testing.T) {
	_ = testlib.Mktmp(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "bar",
				Name:  "bar",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "bar", ctx.Config.ProjectName)
}

func TestEmptyProjectName_DefaultsToGitLabRelease(t *testing.T) {
	_ = testlib.Mktmp(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Release: config.Release{
			GitLab: config.Repo{
				Owner: "bar",
				Name:  "bar",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "bar", ctx.Config.ProjectName)
}

func TestEmptyProjectName_DefaultsToGiteaRelease(t *testing.T) {
	_ = testlib.Mktmp(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Release: config.Release{
			Gitea: config.Repo{
				Owner: "bar",
				Name:  "bar",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "bar", ctx.Config.ProjectName)
}

func TestEmptyProjectName_DefaultsToGoModPath(t *testing.T) {
	_ = testlib.Mktmp(t)
	ctx := testctx.Wrap(t.Context())
	require.NoError(t, exec.CommandContext(t.Context(), "go", "mod", "init", "github.com/foo/bar").Run())
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "bar", ctx.Config.ProjectName)
}

func TestProjectNameWithSkippedRelease(t *testing.T) {
	_ = testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:acme/foo.git")
	require.NoError(t, exec.CommandContext(t.Context(), "go", "mod", "init", "github.com/acme/foo/v2").Run())

	for _, tt := range []struct {
		name        string
		projectName string
		disable     string
		skip        bool
		want        string
	}{
		{name: "release", want: "foo"},
		{name: "skipped", skip: true, want: "v2"},
		{name: "disabled", disable: "true", want: "v2"},
		{name: "release with explicit name", projectName: "foo", want: "foo"},
		{name: "skipped with explicit name", projectName: "foo", skip: true, want: "foo"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				ProjectName: tt.projectName,
				Release:     config.Release{Disable: tt.disable},
			}, testctx.GitHubTokenType)
			if tt.skip {
				skips.Set(ctx, skips.SCMRelease)
			}
			err := release.Pipe{}.Default(ctx)
			if tt.skip || tt.disable == "true" {
				testlib.AssertSkipped(t, err)
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, Pipe{}.Default(ctx))
			require.Equal(t, tt.want, ctx.Config.ProjectName)
		})
	}
}

func TestEmptyProjectName_GoModPathIgnoresToolchainNotice(t *testing.T) {
	dir := testlib.Mktmp(t)
	name := "go"
	content := "#!/bin/sh\necho \"go: downloading go1.27.0 (linux/arm64)\" >&2\necho demo\n"
	if testlib.IsWindows() {
		name = "go.bat"
		content = "@echo off\r\necho go: downloading go1.27.0 (linux/arm64) 1>&2\r\necho demo\r\n"
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	ctx := testctx.Wrap(t.Context())
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "demo", ctx.Config.ProjectName)
}

func TestEmptyProjectName_DefaultsToCargo(t *testing.T) {
	_ = testlib.Mktmp(t)
	ctx := testctx.Wrap(t.Context())
	require.NoError(t, os.WriteFile("Cargo.toml", []byte("[package]\nname = \"bar\""), 0o644))
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "bar", ctx.Config.ProjectName)
}

func TestEmptyProjectName_DefaultsToGitURL(t *testing.T) {
	_ = testlib.Mktmp(t)
	ctx := testctx.Wrap(t.Context())
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "bar", ctx.Config.ProjectName)
}

func TestEmptyProjectName_DefaultsToNonSCMGitURL(t *testing.T) {
	_ = testlib.Mktmp(t)
	ctx := testctx.Wrap(t.Context())
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@myhost.local:bar.git")
	require.EqualError(t, Pipe{}.Default(ctx), "couldn't guess project_name, please add it to your config")
}

func TestEmptyProjectNameAndRelease(t *testing.T) {
	_ = testlib.Mktmp(t)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Release: config.Release{
			GitHub: config.Repo{},
		},
	})

	require.EqualError(t, Pipe{}.Default(ctx), "couldn't guess project_name, please add it to your config")
}
