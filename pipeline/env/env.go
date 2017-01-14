package env

import (
	"errors"
	"os"

	"github.com/goreleaser/releaser/context"
)

var ErrMissingToken = errors.New("Missing GITHUB_TOKEN")

// Pipe for env
type Pipe struct{}

// Name of the pipe
func (Pipe) Name() string {
	return "Env"
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
