// Package env implements the Pipe interface providing validation of
// missing environment variables needed by the release process.
package env

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
	homedir "github.com/mitchellh/go-homedir"
)

// ErrMissingToken indicates an error when GITHUB_TOKEN, GITLAB_TOKEN and GITEA_TOKEN are all missing in the environment.
var ErrMissingToken = errors.New("missing GITHUB_TOKEN, GITLAB_TOKEN and GITEA_TOKEN")

// ErrMultipleTokens indicates that multiple tokens are defined. ATM only one of them if allowed.
// See https://github.com/goreleaser/goreleaser/pull/809
var ErrMultipleTokens = errors.New("multiple tokens defined. Only one is allowed")

// Pipe for env.
type Pipe struct{}

func (Pipe) String() string {
	return "loading environment variables"
}

func setDefaultTokenFiles(ctx *context.Context) {
	env := &ctx.Config.EnvFiles
	if env.GitHubToken == "" {
		env.GitHubToken = "~/.config/goreleaser/github_token"
	}
	if env.GitLabToken == "" {
		env.GitLabToken = "~/.config/goreleaser/gitlab_token"
	}
	if env.GiteaToken == "" {
		env.GiteaToken = "~/.config/goreleaser/gitea_token"
	}
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	templ := tmpl.New(ctx).WithEnvS(os.Environ())
	for i := range ctx.Config.Env {
		env, err := templ.Apply(ctx.Config.Env[i])
		if err != nil {
			return err
		}
		// XXX: this has no risk of panicking because it would already have
		// panicked at `context.go`'s `splitEnv` method.
		// Need to properly handle this at some point.
		parts := strings.SplitN(env, "=", 2)
		ctx.Env[parts[0]] = parts[1]
	}

	setDefaultTokenFiles(ctx)
	githubToken, githubTokenErr := loadEnv("GITHUB_TOKEN", ctx.Config.EnvFiles.GitHubToken)
	gitlabToken, gitlabTokenErr := loadEnv("GITLAB_TOKEN", ctx.Config.EnvFiles.GitLabToken)
	giteaToken, giteaTokenErr := loadEnv("GITEA_TOKEN", ctx.Config.EnvFiles.GiteaToken)

	numOfTokens := 0
	if githubToken != "" {
		numOfTokens++
	}
	if gitlabToken != "" {
		numOfTokens++
	}
	if giteaToken != "" {
		numOfTokens++
	}
	if numOfTokens > 1 {
		return ErrMultipleTokens
	}

	noTokens := githubToken == "" && gitlabToken == "" && giteaToken == ""
	noTokenErrs := githubTokenErr == nil && gitlabTokenErr == nil && giteaTokenErr == nil

	if err := checkErrors(ctx, noTokens, noTokenErrs, gitlabTokenErr, githubTokenErr, giteaTokenErr); err != nil {
		return err
	}

	if gitlabToken != "" {
		log.Debug("token type: gitlab")
		ctx.TokenType = context.TokenTypeGitLab
		ctx.Token = gitlabToken
	}

	if giteaToken != "" {
		log.Debug("token type: gitea")
		ctx.TokenType = context.TokenTypeGitea
		ctx.Token = giteaToken
	}

	if githubToken != "" {
		log.Debug("token type: github")
		ctx.Token = githubToken
	}

	if ctx.TokenType == "" {
		ctx.TokenType = context.TokenTypeGitHub
	}

	return nil
}

func checkErrors(ctx *context.Context, noTokens, noTokenErrs bool, gitlabTokenErr, githubTokenErr, giteaTokenErr error) error {
	if ctx.SkipTokenCheck || ctx.SkipPublish || ctx.Config.Release.Disable {
		return nil
	}

	if noTokens && noTokenErrs {
		return ErrMissingToken
	}

	if gitlabTokenErr != nil {
		return fmt.Errorf("failed to load gitlab token: %w", gitlabTokenErr)
	}

	if githubTokenErr != nil {
		return fmt.Errorf("failed to load github token: %w", githubTokenErr)
	}

	if giteaTokenErr != nil {
		return fmt.Errorf("failed to load gitea token: %w", giteaTokenErr)
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
	defer f.Close()
	bts, _, err := bufio.NewReader(f).ReadLine()
	return string(bts), err
}
