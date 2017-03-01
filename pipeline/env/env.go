package env

import (
	"errors"
	"os"

	"github.com/goreleaser/goreleaser/context"
)

// Pipe for env
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Loading environment variables"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) (err error) {
	ctx.Token = os.Getenv("GITHUB_TOKEN")
	return
}
