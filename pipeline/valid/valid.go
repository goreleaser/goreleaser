package valid

import (
	"errors"

	"github.com/goreleaser/releaser/context"
)

// Pipe for brew deployment
type Pipe struct{}

// Name of the pipe
func (Pipe) Description() string {
	return "Validating configuration..."
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) (err error) {
	if ctx.Config.BinaryName == "" {
		return errors.New("missing binary_name")
	}
	return
}
