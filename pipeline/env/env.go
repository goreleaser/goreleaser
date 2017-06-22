// Package env implements the Pipe interface providing validation of
// missing environment variables needed by the release process.
package env

import (
	"errors"
	"os"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/context"
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
		log.Warn("github token not validated because publishing has been disabled")
		return nil
	}
	if !ctx.Validate {
		log.Warn("skipped validations because --skip-validate is set")
		return nil
	}
	if ctx.Token == "" {
		return ErrMissingToken
	}
	return
}
