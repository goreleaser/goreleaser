// Package upload provides a Pipe that push using HTTP.
package upload

import (
	"fmt"
	h "net/http"

	"github.com/goreleaser/goreleaser/internal/http"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe for http publishing.
type Pipe struct{}

// String returns the description of the pipe.
func (Pipe) String() string                 { return "http upload" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.Uploads) == 0 }

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	return http.Defaults(ctx.Config.Uploads)
}

// Publish artifacts.
func (Pipe) Publish(ctx *context.Context) error {
	// Check requirements for every instance we have configured.
	// If not fulfilled, we can skip this pipeline
	for _, instance := range ctx.Config.Uploads {
		instance := instance
		if skip := http.CheckConfig(ctx, &instance, "upload"); skip != nil {
			return pipe.Skip(skip.Error())
		}
	}

	return http.Upload(ctx, ctx.Config.Uploads, "upload", func(res *h.Response) error {
		if c := res.StatusCode; c < 200 || 299 < c {
			return fmt.Errorf("unexpected http response status: %s", res.Status)
		}
		return nil
	})
}
