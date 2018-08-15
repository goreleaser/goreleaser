// Package fpm implements the Pipe interface providing FPM bindings.
package fpm

import (
	"github.com/goreleaser/goreleaser/internal/deprecate"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe for fpm packaging
type Pipe struct{}

func (Pipe) String() string {
	return "creating Linux packages with fpm"
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	if len(ctx.Config.FPM.Formats) > 0 && len(ctx.Config.NFPM.Formats) == 0 {
		deprecate.Notice("fpm")
		ctx.Config.NFPM = ctx.Config.FPM
	}
	return nil
}
