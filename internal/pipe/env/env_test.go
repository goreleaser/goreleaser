package env

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestDefault(t *testing.T) {
	t.Run("empty config", func(tt *testing.T) {
		ctx := context.New(config.Project{})
		assert.NoError(t, Pipe{}.Default(ctx))
		assert.Equal(t, "~/.config/goreleaser/github_token", ctx.Config.EnvFiles.GitHubToken)
	})
	t.Run("custom config config", func(tt *testing.T) {
		cfg := "what"
		ctx := context.New(config.Project{
			EnvFiles: config.EnvFiles{
				GitHubToken: cfg,
			},
		})
		assert.NoError(t, Pipe{}.Default(ctx))
		assert.Equal(t, cfg, ctx.Config.EnvFiles.GitHubToken)
	})
}

func TestValidEnv(t *testing.T) {
	assert.NoError(t, os.Setenv("GITHUB_TOKEN", "asdf"))
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(t, Pipe{}.Run(ctx))
}

func TestInvalidEnv(t *testing.T) {
	assert.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.Error(t, Pipe{}.Run(ctx))
}

func TestEmptyFileEnv(t *testing.T) {
	assert.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.Error(t, Pipe{}.Run(ctx))
}

func TestEmptyEnvFile(t *testing.T) {
	assert.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
	f, err := ioutil.TempFile("", "token")
	assert.NoError(t, err)
	assert.NoError(t, os.Chmod(f.Name(), 0377))
	var ctx = &context.Context{
		Config: config.Project{
			EnvFiles: config.EnvFiles{
				GitHubToken: f.Name(),
			},
		},
	}
	assert.EqualError(t, Pipe{}.Run(ctx), fmt.Sprintf("failed to load github token: open %s: permission denied", f.Name()))
}

func TestInvalidEnvChecksSkipped(t *testing.T) {
	assert.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
	var ctx = &context.Context{
		Config:      config.Project{},
		SkipPublish: true,
	}
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestInvalidEnvReleaseDisabled(t *testing.T) {
	assert.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
	var ctx = &context.Context{
		Config: config.Project{
			Release: config.Release{
				Disable: true,
			},
		},
	}
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestLoadEnv(t *testing.T) {
	t.Run("env exists", func(tt *testing.T) {
		var env = "SUPER_SECRET_ENV"
		assert.NoError(tt, os.Setenv(env, "1"))
		v, err := loadEnv(env, "nope")
		assert.NoError(tt, err)
		assert.Equal(tt, "1", v)
	})
	t.Run("env file exists", func(tt *testing.T) {
		var env = "SUPER_SECRET_ENV_NOPE"
		assert.NoError(tt, os.Unsetenv(env))
		f, err := ioutil.TempFile("", "token")
		assert.NoError(t, err)
		fmt.Fprintf(f, "123")
		v, err := loadEnv(env, f.Name())
		assert.NoError(tt, err)
		assert.Equal(tt, "123", v)
	})
	t.Run("env file with an empty line at the end", func(tt *testing.T) {
		var env = "SUPER_SECRET_ENV_NOPE"
		assert.NoError(tt, os.Unsetenv(env))
		f, err := ioutil.TempFile("", "token")
		assert.NoError(t, err)
		fmt.Fprintf(f, "123\n")
		v, err := loadEnv(env, f.Name())
		assert.NoError(tt, err)
		assert.Equal(tt, "123", v)
	})
	t.Run("env file is not readable", func(tt *testing.T) {
		var env = "SUPER_SECRET_ENV_NOPE"
		assert.NoError(tt, os.Unsetenv(env))
		f, err := ioutil.TempFile("", "token")
		assert.NoError(t, err)
		fmt.Fprintf(f, "123")
		err = os.Chmod(f.Name(), 0377)
		assert.NoError(tt, err)
		v, err := loadEnv(env, f.Name())
		assert.EqualError(tt, err, fmt.Sprintf("open %s: permission denied", f.Name()))
		assert.Equal(tt, "", v)
	})
}
