package env

import (
	"errors"
	"os"

	"github.com/goreleaser/goreleaser/context"
)

// ErrMissingToken indicates an error when GITHUB_TOKEN is missing in the environment
var ErrMissingToken = errors.New("Missing GITHUB_TOKEN")

// Pipe for env
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Loading data from environment variables..."
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) (err error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return ErrMissingToken
	}
	ctx.Token = &token
	return
}
