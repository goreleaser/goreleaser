package project

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
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

func TestEmptyProjectNameAndRelease(t *testing.T) {
	_ = testlib.Mktmp(t)
	ctx := testctx.NewWithCfg(config.Project{
		Release: config.Release{
			GitHub: config.Repo{},
		},
	})
	require.EqualError(t, Pipe{}.Default(ctx), "couldn't guess project_name, please add it to your config")
}
