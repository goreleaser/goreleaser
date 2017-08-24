// Package env implements the Pipe interface providing validation of
// missing environment variables needed by the release process.
package env

import (
	"errors"
	"os"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline"
)

// ErrMissingToken indicates an error when GITHUB_TOKEN is missing in the environment
var ErrMissingToken = errors.New("missing GITHUB_TOKEN")

// Pipe for env
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Loading environment variables"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) (err error) {
	ctx.Token = os.Getenv("GITHUB_TOKEN")
	if !ctx.Publish {
		return pipeline.Skip("publishing is disabled")
	}
	if !ctx.Validate {
		return pipeline.Skip("--skip-validate is set")
	}
	if ctx.Token == "" {
		return ErrMissingToken
	}
	return
}
