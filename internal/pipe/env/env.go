// Package env implements the Pipe interface providing validation of
// missing environment variables needed by the release process.
package env

import (
	"bufio"
	"os"

	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/pkg/context"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

// ErrMissingToken indicates an error when GITHUB_TOKEN and GITLAB_TOKEN are missing in the environment
var ErrMissingToken = errors.New("missing GITHUB_TOKEN and GITLAB_TOKEN")

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
	if env.GitLabToken == "" {
		env.GitLabToken = "~/.config/goreleaser/gitlab_token"
	}
	return nil
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	githubToken, githubTokenErr := loadEnv("GITHUB_TOKEN", ctx.Config.EnvFiles.GitHubToken)
	gitlabToken, gitlabTokenErr := loadEnv("GITLAB_TOKEN", ctx.Config.EnvFiles.GitLabToken)
	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}
	if ctx.Config.Release.Disable {
		return pipe.Skip("release pipe is disabled")
	}
	if githubToken == "" && gitlabToken == "" && githubTokenErr == nil && gitlabTokenErr == nil {
		return ErrMissingToken
	}
	if githubTokenErr != nil {
		return errors.Wrap(githubTokenErr, "failed to load github token")
	}
	if gitlabTokenErr != nil {
		return errors.Wrap(gitlabTokenErr, "failed to load gitlab token")
	}
	if githubToken != "" {
		ctx.StorageToken = githubToken
		ctx.StorageType = context.StorageGitHub
	}
	if gitlabToken != "" {
		ctx.StorageToken = gitlabToken
		ctx.StorageType = context.StorageGitLab
	}

	return nil
}

func loadEnv(env, path string) (string, error) {
	val := os.Getenv(env)
	if val != "" {
		return val, nil
	}
	path, err := homedir.Expand(path)
	if err != nil {
		return "", err
	}
	f, err := os.Open(path) // #nosec
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	bts, _, err := bufio.NewReader(f).ReadLine()
	return string(bts), err
}
