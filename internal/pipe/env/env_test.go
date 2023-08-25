package env

import (
	"fmt"
	"os"
	"syscall"
	"testing"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	restores := map[string]string{}
	for _, key := range []string{"GITHUB_TOKEN", "GITEA_TOKEN", "GITLAB_TOKEN"} {
		prevValue, ok := os.LookupEnv(key)
		if ok {
			_ = os.Unsetenv(key)
			restores[key] = prevValue
		}
	}

	code := m.Run()

	for k, v := range restores {
		_ = os.Setenv(k, v)
	}

	os.Exit(code)
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSetDefaultTokenFiles(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		ctx := testctx.New()
		setDefaultTokenFiles(ctx)
		require.Equal(t, "~/.config/goreleaser/github_token", ctx.Config.EnvFiles.GitHubToken)
		require.Equal(t, "~/.config/goreleaser/gitlab_token", ctx.Config.EnvFiles.GitLabToken)
		require.Equal(t, "~/.config/goreleaser/gitea_token", ctx.Config.EnvFiles.GiteaToken)
	})
	t.Run("custom config config", func(t *testing.T) {
		cfg := "what"
		ctx := testctx.NewWithCfg(config.Project{
			EnvFiles: config.EnvFiles{
				GitHubToken: cfg,
			},
		})
		setDefaultTokenFiles(ctx)
		require.Equal(t, cfg, ctx.Config.EnvFiles.GitHubToken)
	})
	t.Run("templates", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			ProjectName: "foobar",
			Env: []string{
				"FOO=FOO_{{ .Env.BAR }}",
				"FOOBAR={{.ProjectName}}",
				"EMPTY_VAL=",
			},
		})
		ctx.Env["FOOBAR"] = "old foobar"
		t.Setenv("BAR", "lebar")
		t.Setenv("GITHUB_TOKEN", "fake")
		require.NoError(t, Pipe{}.Run(ctx))
		require.Equal(t, "FOO_lebar", ctx.Env["FOO"])
		require.Equal(t, "foobar", ctx.Env["FOOBAR"])
		require.Equal(t, "", ctx.Env["EMPTY_VAL"])
	})

	t.Run("template error", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Env: []string{
				"FOO={{ .Asss }",
			},
		})
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})

	t.Run("no token", func(t *testing.T) {
		ctx := testctx.New()
		require.EqualError(t, Pipe{}.Run(ctx), ErrMissingToken.Error())
	})
}

func TestForceToken(t *testing.T) {
	t.Run("github", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "fake")
		t.Setenv("GORELEASER_FORCE_TOKEN", "github")
		ctx := testctx.New()
		require.NoError(t, Pipe{}.Run(ctx))
		require.Equal(t, context.TokenTypeGitHub, ctx.TokenType)
	})
	t.Run("gitlab", func(t *testing.T) {
		t.Setenv("GITLAB_TOKEN", "fake")
		t.Setenv("GORELEASER_FORCE_TOKEN", "gitlab")
		ctx := testctx.New()
		require.NoError(t, Pipe{}.Run(ctx))
		require.Equal(t, context.TokenTypeGitLab, ctx.TokenType)
	})
	t.Run("gitea", func(t *testing.T) {
		t.Setenv("GITEA_TOKEN", "fake")
		t.Setenv("GORELEASER_FORCE_TOKEN", "gitea")
		ctx := testctx.New()
		require.NoError(t, Pipe{}.Run(ctx))
		require.Equal(t, context.TokenTypeGitea, ctx.TokenType)
	})
}

func TestValidGithubEnv(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "asdf")
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "asdf", ctx.Token)
	require.Equal(t, context.TokenTypeGitHub, ctx.TokenType)
}

func TestValidGitlabEnv(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "qwertz")
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "qwertz", ctx.Token)
	require.Equal(t, context.TokenTypeGitLab, ctx.TokenType)
}

func TestValidGiteaEnv(t *testing.T) {
	t.Setenv("GITEA_TOKEN", "token")
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "token", ctx.Token)
	require.Equal(t, context.TokenTypeGitea, ctx.TokenType)
}

func TestInvalidEnv(t *testing.T) {
	ctx := testctx.New()
	require.Error(t, Pipe{}.Run(ctx))
	require.EqualError(t, Pipe{}.Run(ctx), ErrMissingToken.Error())
}

func TestMultipleEnvTokens(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "asdf")
	t.Setenv("GITLAB_TOKEN", "qwertz")
	t.Setenv("GITEA_TOKEN", "token")
	ctx := testctx.New()
	require.Error(t, Pipe{}.Run(ctx))
	require.EqualError(t, Pipe{}.Run(ctx), "multiple tokens found, but only one is allowed: GITHUB_TOKEN, GITLAB_TOKEN, GITEA_TOKEN\n\nLearn more at https://goreleaser.com/errors/multiple-tokens\n")
}

func TestMultipleEnvTokensForce(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "asdf")
	t.Setenv("GITLAB_TOKEN", "qwertz")
	t.Setenv("GITEA_TOKEN", "token")
	ctx := testctx.New()
	require.Error(t, Pipe{}.Run(ctx))
	require.EqualError(t, Pipe{}.Run(ctx), "multiple tokens found, but only one is allowed: GITHUB_TOKEN, GITLAB_TOKEN, GITEA_TOKEN\n\nLearn more at https://goreleaser.com/errors/multiple-tokens\n")
}

func TestEmptyGithubFileEnv(t *testing.T) {
	ctx := testctx.New()
	require.Error(t, Pipe{}.Run(ctx))
}

func TestEmptyGitlabFileEnv(t *testing.T) {
	ctx := testctx.New()
	require.Error(t, Pipe{}.Run(ctx))
}

func TestEmptyGiteaFileEnv(t *testing.T) {
	ctx := testctx.New()
	require.Error(t, Pipe{}.Run(ctx))
}

func TestEmptyGithubEnvFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "token")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Chmod(f.Name(), 0o377))
	ctx := testctx.NewWithCfg(config.Project{
		EnvFiles: config.EnvFiles{
			GitHubToken: f.Name(),
		},
	})
	err = Pipe{}.Run(ctx)
	require.ErrorIs(t, err, syscall.EACCES)
	require.Contains(t, err.Error(), "failed to load github token")
}

func TestEmptyGitlabEnvFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "token")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Chmod(f.Name(), 0o377))
	ctx := testctx.NewWithCfg(config.Project{
		EnvFiles: config.EnvFiles{
			GitLabToken: f.Name(),
		},
	})
	err = Pipe{}.Run(ctx)
	require.ErrorIs(t, err, syscall.EACCES)
	require.Contains(t, err.Error(), "failed to load gitlab token")
}

func TestEmptyGiteaEnvFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "token")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Chmod(f.Name(), 0o377))
	ctx := testctx.NewWithCfg(config.Project{
		EnvFiles: config.EnvFiles{
			GiteaToken: f.Name(),
		},
	})
	err = Pipe{}.Run(ctx)
	require.ErrorIs(t, err, syscall.EACCES)
	require.Contains(t, err.Error(), "failed to load gitea token")
}

func TestInvalidEnvChecksSkipped(t *testing.T) {
	ctx := testctx.New(testctx.SkipPublish)
	require.NoError(t, Pipe{}.Run(ctx))
}

func TestInvalidEnvReleaseDisabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Env: []string{},
			Release: config.Release{
				Disable: "true",
			},
		})
		require.NoError(t, Pipe{}.Run(ctx))
	})

	t.Run("tmpl true", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Env: []string{"FOO=true"},
			Release: config.Release{
				Disable: "{{ .Env.FOO }}",
			},
		})
		require.NoError(t, Pipe{}.Run(ctx))
	})

	t.Run("tmpl false", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Env: []string{"FOO=true"},
			Release: config.Release{
				Disable: "{{ .Env.FOO }}-nope",
			},
		})
		require.EqualError(t, Pipe{}.Run(ctx), ErrMissingToken.Error())
	})

	t.Run("tmpl error", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Release: config.Release{
				Disable: "{{ .Env.FOO }}",
			},
		})
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})
}

func TestLoadEnv(t *testing.T) {
	const env = "SUPER_SECRET_ENV_NOPE"

	t.Run("env exists", func(t *testing.T) {
		t.Setenv(env, "1")
		v, err := loadEnv(env, "nope")
		require.NoError(t, err)
		require.Equal(t, "1", v)
	})
	t.Run("env file exists", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "token")
		require.NoError(t, err)
		fmt.Fprintf(f, "123")
		require.NoError(t, f.Close())
		v, err := loadEnv(env, f.Name())
		require.NoError(t, err)
		require.Equal(t, "123", v)
	})
	t.Run("env file with an empty line at the end", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "token")
		require.NoError(t, err)
		fmt.Fprintf(f, "123\n")
		require.NoError(t, f.Close())
		v, err := loadEnv(env, f.Name())
		require.NoError(t, err)
		require.Equal(t, "123", v)
	})
	t.Run("env file is not readable", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "token")
		require.NoError(t, err)
		fmt.Fprintf(f, "123")
		require.NoError(t, f.Close())
		err = os.Chmod(f.Name(), 0o377)
		require.NoError(t, err)
		v, err := loadEnv(env, f.Name())
		require.EqualError(t, err, fmt.Sprintf("open %s: permission denied", f.Name()))
		require.Equal(t, "", v)
	})
}
