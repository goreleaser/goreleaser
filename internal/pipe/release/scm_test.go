package release

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/git"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestSetupGitLab(t *testing.T) {
	t.Run("no repo", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		require.NoError(t, setupGitLab(ctx))
		repo, err := git.ExtractRepoFromConfig(ctx)
		require.NoError(t, err)
		require.Equal(t, repo.Owner, ctx.Config.Release.GitLab.Owner)
		require.Equal(t, repo.Name, ctx.Config.Release.GitLab.Name)
	})

	t.Run("with templates", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Env: []string{"NAME=foo", "OWNER=bar"},
			GitLabURLs: config.GitLabURLs{
				Download: "https://{{ .Env.OWNER }}/download",
			},
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
		require.Equal(t, "https://bar/download/bar/foo/-/releases/", ctx.ReleaseURL)
	})

	t.Run("with invalid templates", func(t *testing.T) {
		t.Run("owner", func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
		ctx := testctx.Wrap(t.Context())
		require.NoError(t, setupGitea(ctx))
		require.Equal(t, "goreleaser", ctx.Config.Release.Gitea.Name)
	})

	t.Run("with templates", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Env: []string{"NAME=foo", "OWNER=bar"},
			GiteaURLs: config.GiteaURLs{
				Download: "https://{{ .Env.OWNER }}/download",
			},
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
		require.Equal(t, "https://bar/download/bar/foo/releases/tag/", ctx.ReleaseURL)
	})

	t.Run("with invalid templates", func(t *testing.T) {
		t.Run("owner", func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
		ctx := testctx.Wrap(t.Context())
		require.NoError(t, setupGitHub(ctx))
		require.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Name)
	})

	t.Run("with templates", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Env: []string{"NAME=foo", "OWNER=bar"},
			GitHubURLs: config.GitHubURLs{
				Download: "https://{{ .Env.OWNER }}/download",
			},
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
		require.Equal(t, "https://bar/download/bar/foo/releases/tag/", ctx.ReleaseURL)
	})

	t.Run("with invalid templates", func(t *testing.T) {
		t.Run("owner", func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

func TestSetupPreservesToken(t *testing.T) {
	for name, tt := range map[string]struct {
		remote string
		cfg    config.Project
		setup  func(*context.Context) error
		repo   func(*context.Context) config.Repo
	}{
		"github": {
			remote: "git@github.com:goreleaser/goreleaser.git",
			cfg: config.Project{
				Release: config.Release{
					GitHub: config.Repo{Token: "{{ .Env.RELEASE_TOKEN }}"},
				},
			},
			setup: setupGitHub,
			repo:  func(ctx *context.Context) config.Repo { return ctx.Config.Release.GitHub },
		},
		"gitlab": {
			remote: "git@gitlab.com:goreleaser/goreleaser.git",
			cfg: config.Project{
				Release: config.Release{
					GitLab: config.Repo{Token: "{{ .Env.RELEASE_TOKEN }}"},
				},
			},
			setup: setupGitLab,
			repo:  func(ctx *context.Context) config.Repo { return ctx.Config.Release.GitLab },
		},
		"gitea": {
			remote: "git@gitea.example.com:goreleaser/goreleaser.git",
			cfg: config.Project{
				Release: config.Release{
					Gitea: config.Repo{Token: "{{ .Env.RELEASE_TOKEN }}"},
				},
			},
			setup: setupGitea,
			repo:  func(ctx *context.Context) config.Repo { return ctx.Config.Release.Gitea },
		},
	} {
		t.Run(name+"/inferred", func(t *testing.T) {
			testlib.Mktmp(t)
			testlib.GitInit(t)
			testlib.GitRemoteAdd(t, tt.remote)

			ctx := testctx.WrapWithCfg(t.Context(), tt.cfg)
			require.NoError(t, tt.setup(ctx))
			repo := tt.repo(ctx)
			require.Equal(t, "goreleaser", repo.Owner)
			require.Equal(t, "goreleaser", repo.Name)
			require.Equal(t, "{{ .Env.RELEASE_TOKEN }}", repo.Token)
		})
	}

	for name, tt := range map[string]struct {
		cfg   config.Project
		setup func(*context.Context) error
		repo  func(*context.Context) config.Repo
	}{
		"github": {
			cfg: config.Project{
				Release: config.Release{
					GitHub: config.Repo{Owner: "owner", Name: "name", Token: "{{ .Env.RELEASE_TOKEN }}"},
				},
			},
			setup: setupGitHub,
			repo:  func(ctx *context.Context) config.Repo { return ctx.Config.Release.GitHub },
		},
		"gitlab": {
			cfg: config.Project{
				Release: config.Release{
					GitLab: config.Repo{Owner: "owner", Name: "name", Token: "{{ .Env.RELEASE_TOKEN }}"},
				},
			},
			setup: setupGitLab,
			repo:  func(ctx *context.Context) config.Repo { return ctx.Config.Release.GitLab },
		},
		"gitea": {
			cfg: config.Project{
				Release: config.Release{
					Gitea: config.Repo{Owner: "owner", Name: "name", Token: "{{ .Env.RELEASE_TOKEN }}"},
				},
			},
			setup: setupGitea,
			repo:  func(ctx *context.Context) config.Repo { return ctx.Config.Release.Gitea },
		},
	} {
		t.Run(name+"/explicit", func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), tt.cfg)
			require.NoError(t, tt.setup(ctx))
			repo := tt.repo(ctx)
			require.Equal(t, "owner", repo.Owner)
			require.Equal(t, "name", repo.Name)
			require.Equal(t, "{{ .Env.RELEASE_TOKEN }}", repo.Token)
		})
	}
}
