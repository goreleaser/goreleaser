// Package fpm implements the Pipe interface providing FPM bindings.
package fpm

import (
	"github.com/pkg/errors"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/deprecate"
)

// ErrNoFPM is shown when fpm cannot be found in $PATH
var ErrNoFPM = errors.New("fpm not present in $PATH")

const (
	defaultNameTemplate = "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"
	// path to gnu-tar on macOS when installed with homebrew
	gnuTarPath = "/usr/local/opt/gnu-tar/libexec/gnubin"
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
