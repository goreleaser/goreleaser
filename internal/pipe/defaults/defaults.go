// Package defaults implements the Pipe interface providing default values
// for missing configuration.
package defaults

import (
	"cmp"
	"fmt"
	"strings"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/middleware/errhandler"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/goreleaser/goreleaser/v2/pkg/defaults"
)

// Pipe that sets the defaults.
type Pipe struct{}

func (Pipe) String() string { return "setting defaults" }

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	if ctx.Config.GitHubURLs.Download == "" {
		ctx.Config.GitHubURLs.Download = client.DefaultGitHubDownloadURL
	}
	if ctx.Config.GitLabURLs.Download == "" {
		ctx.Config.GitLabURLs.Download = client.DefaultGitLabDownloadURL
	}
	if ctx.Config.GiteaURLs.Download == "" {
		apiURL, err := tmpl.New(ctx).Apply(ctx.Config.GiteaURLs.API)
		if err != nil {
			return fmt.Errorf("templating Gitea API URL: %w", err)
		}

		ctx.Config.GiteaURLs.Download = strings.TrimSuffix(strings.ReplaceAll(apiURL, "/api/v1", ""), "/")
	}

	ctx.Config.Retry.Attempts = cmp.Or(ctx.Config.Retry.Attempts, 10)
	ctx.Config.Retry.Delay = cmp.Or(ctx.Config.Retry.Delay, 10*time.Second)
	ctx.Config.Retry.MaxDelay = cmp.Or(ctx.Config.Retry.MaxDelay, 5*time.Minute)

	for _, defaulter := range defaults.Defaulters {
		if err := errhandler.Handle(defaulter.Default)(ctx); err != nil {
			return fmt.Errorf("%s: %w", defaulter.String(), err)
		}
	}
	return nil
}
