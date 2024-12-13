package project

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
)

func TestCustomProjectName(t *testing.T) {
	_ = testlib.Mktmp(t)
	ctx := testctx.NewWithCfg(config.Project{
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
	ctx := testctx.NewWithCfg(config.Project{
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
	ctx := testctx.NewWithCfg(config.Project{
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
	ctx := testctx.NewWithCfg(config.Project{
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
	ctx := testctx.New()
	require.NoError(t, exec.Command("go", "mod", "init", "github.com/foo/bar").Run())
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "bar", ctx.Config.ProjectName)
}

func TestEmptyProjectName_DefaultsToCargo(t *testing.T) {
	_ = testlib.Mktmp(t)
	ctx := testctx.New()
	require.NoError(t, os.WriteFile("Cargo.toml", []byte("[package]\nname = \"bar\""), 0o644))
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "bar", ctx.Config.ProjectName)
}

func TestEmptyProjectName_DefaultsToGitURL(t *testing.T) {
	_ = testlib.Mktmp(t)
	ctx := testctx.New()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "bar", ctx.Config.ProjectName)
}

func TestEmptyProjectName_DefaultsToNonSCMGitURL(t *testing.T) {
	_ = testlib.Mktmp(t)
	ctx := testctx.New()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@myhost.local:bar.git")
	require.EqualError(t, Pipe{}.Default(ctx), "couldn't guess project_name, please add it to your config")
}

func TestEmptyProjectNameAndRelease(t *testing.T) {
	_ = testlib.Mktmp(t)
	ctx := testctx.NewWithCfg(config.Project{
		Release: config.Release{
			GitHub: config.Repo{},
		},
	})
	require.EqualError(t, Pipe{}.Default(ctx), "couldn't guess project_name, please add it to your config")
}
