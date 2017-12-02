package pipeline

import (
	"fmt"

	"github.com/goreleaser/goreleaser/context"
)

// Defaulter interface
type Defaulter interface {
	fmt.Stringer

	// Default sets the configuration defaults
	Default(ctx *context.Context) error
}
