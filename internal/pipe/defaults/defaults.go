// Package defaults implements the Pipe interface providing default values
// for missing configuration.
package defaults

import (
	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/goreleaser/goreleaser/pkg/defaults"
)

// Pipe that sets the defaults
type Pipe struct{}

func (Pipe) String() string {
	return "setting defaults"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	if ctx.Config.Dist == "" {
		ctx.Config.Dist = "dist"
	}
	if ctx.Config.GitHubURLs.Download == "" {
		ctx.Config.GitHubURLs.Download = "https://github.com"
	}
	for _, defaulter := range defaults.Defaulters {
		log.Debug(defaulter.String())
		if err := defaulter.Default(ctx); err != nil {
			return err
		}
	}
	return nil
}
