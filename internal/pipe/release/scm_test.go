package release

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestSetupGitLab(t *testing.T) {
	t.Run("no repo", func(t *testing.T) {
		ctx := context.New(config.Project{})

		require.NoError(t, setupGitLab(ctx))
		require.Equal(t, "goreleaser", ctx.Config.Release.GitLab.Owner)
		require.Equal(t, "goreleaser", ctx.Config.Release.GitLab.Name)
	})

	t.Run("with templates", func(t *testing.T) {
		ctx := context.New(config.Project{
			Env: []string{"NAME=foo", "OWNER=bar"},
			Release: config.Release{
				GitLab: config.Repo{
					Owner: "{{.Env.OWNER}}",
					Name:  "{{.Env.NAME}}",
				},
			},
		})

		require.NoError(t, setupGitLab(ctx))
		require.Equal(t, "bar", ctx.Config.Release.GitLab.Owner)
		require.Equal(t, "foo", ctx.Config.Release.GitLab.Name)
	})

	t.Run("with invalid templates", func(t *testing.T) {
		t.Run("owner", func(t *testing.T) {
			ctx := context.New(config.Project{
				Release: config.Release{
					GitLab: config.Repo{
						Name:  "foo",
						Owner: "{{.Env.NOPE}}",
					},
				},
			})

			require.Error(t, setupGitLab(ctx))
		})

		t.Run("name", func(t *testing.T) {
			ctx := context.New(config.Project{
				Release: config.Release{
					GitLab: config.Repo{
						Name: "{{.Env.NOPE}}",
					},
				},
			})

			require.Error(t, setupGitLab(ctx))
		})
	})
}

func TestSetupGitea(t *testing.T) {
	t.Run("no repo", func(t *testing.T) {
		ctx := context.New(config.Project{})

		require.NoError(t, setupGitea(ctx))
		require.Equal(t, "goreleaser", ctx.Config.Release.Gitea.Owner)
		require.Equal(t, "goreleaser", ctx.Config.Release.Gitea.Name)
	})

	t.Run("with templates", func(t *testing.T) {
		ctx := context.New(config.Project{
			Env: []string{"NAME=foo", "OWNER=bar"},
			Release: config.Release{
				Gitea: config.Repo{
					Owner: "{{.Env.OWNER}}",
					Name:  "{{.Env.NAME}}",
				},
			},
		})

		require.NoError(t, setupGitea(ctx))
		require.Equal(t, "bar", ctx.Config.Release.Gitea.Owner)
		require.Equal(t, "foo", ctx.Config.Release.Gitea.Name)
	})

	t.Run("with invalid templates", func(t *testing.T) {
		t.Run("owner", func(t *testing.T) {
			ctx := context.New(config.Project{
				Release: config.Release{
					Gitea: config.Repo{
						Name:  "foo",
						Owner: "{{.Env.NOPE}}",
					},
				},
			})

			require.Error(t, setupGitea(ctx))
		})

		t.Run("name", func(t *testing.T) {
			ctx := context.New(config.Project{
				Release: config.Release{
					Gitea: config.Repo{
						Name: "{{.Env.NOPE}}",
					},
				},
			})

			require.Error(t, setupGitea(ctx))
		})
	})
}

func TestSetupGitHub(t *testing.T) {
	t.Run("no repo", func(t *testing.T) {
		ctx := context.New(config.Project{})

		require.NoError(t, setupGitHub(ctx))
		require.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Owner)
		require.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Name)
	})

	t.Run("with templates", func(t *testing.T) {
		ctx := context.New(config.Project{
			Env: []string{"NAME=foo", "OWNER=bar"},
			Release: config.Release{
				GitHub: config.Repo{
					Owner: "{{.Env.OWNER}}",
					Name:  "{{.Env.NAME}}",
				},
			},
		})

		require.NoError(t, setupGitHub(ctx))
		require.Equal(t, "bar", ctx.Config.Release.GitHub.Owner)
		require.Equal(t, "foo", ctx.Config.Release.GitHub.Name)
	})

	t.Run("with invalid templates", func(t *testing.T) {
		t.Run("owner", func(t *testing.T) {
			ctx := context.New(config.Project{
				Release: config.Release{
					GitHub: config.Repo{
						Name:  "foo",
						Owner: "{{.Env.NOPE}}",
					},
				},
			})

			require.Error(t, setupGitHub(ctx))
		})

		t.Run("name", func(t *testing.T) {
			ctx := context.New(config.Project{
				Release: config.Release{
					GitHub: config.Repo{
						Name: "{{.Env.NOPE}}",
					},
				},
			})

			require.Error(t, setupGitHub(ctx))
		})
	})
}
