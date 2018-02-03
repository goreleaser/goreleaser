// Package env implements the Pipe interface providing validation of
// missing environment variables needed by the release process.
package env

import (
	"io/ioutil"
	"os"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

// ErrMissingToken indicates an error when GITHUB_TOKEN is missing in the environment
var ErrMissingToken = errors.New("missing GITHUB_TOKEN")

// Pipe for env
type Pipe struct{}

func (Pipe) String() string {
	return "loading environment variables"
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	var env = &ctx.Config.EnvFiles
	if env.GitHubToken == "" {
		env.GitHubToken = "~/.config/goreleaser/github_token"
	}
	return nil
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	token, err := loadEnv("GITHUB_TOKEN", ctx.Config.EnvFiles.GitHubToken)
	ctx.Token = token
	if !ctx.Publish {
		return pipeline.Skip("publishing is disabled")
	}
	if !ctx.Validate {
		return pipeline.Skip("--skip-validate is set")
	}
	if ctx.Token == "" && err == nil {
		return ErrMissingToken
	}
	if err != nil {
		return errors.Wrap(err, "failed to load github token")
	}
	return nil
}

func loadEnv(env, path string) (string, error) {
	val := os.Getenv(env)
	if val == "" {
		path, err := homedir.Expand(path)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return "", nil
		}
		bts, err := ioutil.ReadFile(path)
		if err != nil {
			return "", err
		}
		val = string(bts)
	}
	return val, nil
}
