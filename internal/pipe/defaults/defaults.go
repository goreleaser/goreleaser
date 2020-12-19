// Package defaults implements the Pipe interface providing default values
// for missing configuration.
package defaults

import (
	"strings"

	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/middleware"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/goreleaser/goreleaser/pkg/defaults"
)

// Pipe that sets the defaults.
type Pipe struct{}

func (Pipe) String() string {
	return "setting defaults"
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	if ctx.Config.Dist == "" {
		ctx.Config.Dist = "dist"
	}
	if ctx.Config.GitHubURLs.Download == "" {
		ctx.Config.GitHubURLs.Download = client.DefaultGitHubDownloadURL
	}
	if ctx.Config.GitLabURLs.Download == "" {
		ctx.Config.GitLabURLs.Download = client.DefaultGitLabDownloadURL
	}
	if ctx.Config.GiteaURLs.Download == "" {
		ctx.Config.GiteaURLs.Download = strings.ReplaceAll(ctx.Config.GiteaURLs.API, "/api/v1", "")
	}
	for _, defaulter := range defaults.Defaulters {
		if err := middleware.Logging(
			defaulter.String(),
			middleware.ErrHandler(defaulter.Default),
			middleware.ExtraPadding,
		)(ctx); err != nil {
			return err
		}
	}
	return nil
}
