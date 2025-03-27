// Package dist provides checks to make sure the dist directory is always
// empty.
package dist

import (
	"fmt"
	"os"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// CleanPipe cleans the distribution directory.
type CleanPipe struct{}

func (CleanPipe) Skip(ctx *context.Context) bool { return !ctx.Clean }
func (CleanPipe) String() string                 { return "cleaning distribution directory" }
func (CleanPipe) Run(ctx *context.Context) error {
	// here we are setting a default outside a Default method...
	// this is needed because when this run, the defaults are not set yet
	// there's no good way of handling this...
	_ = Pipe{}.Default(ctx)
	return os.RemoveAll(ctx.Config.Dist)
}

// Pipe for dist.
type Pipe struct{}

func (Pipe) String() string { return "ensuring distribution directory" }
func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Dist == "" {
		ctx.Config.Dist = "dist"
	}
	return nil
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	_, err := os.Stat(ctx.Config.Dist)
	if os.IsNotExist(err) {
		log.Debugf("%s doesn't exist, creating empty directory", ctx.Config.Dist)
		return mkdir(ctx)
	}
	files, err := os.ReadDir(ctx.Config.Dist)
	if err != nil {
		return err
	}
	if len(files) != 0 {
		log.Debugf("there are %d files on %s", len(files), ctx.Config.Dist)
		return fmt.Errorf(
			"%s is not empty, remove it before running goreleaser or use the --clean flag",
			ctx.Config.Dist,
		)
	}
	log.Debugf("%s is empty", ctx.Config.Dist)
	return mkdir(ctx)
}

func mkdir(ctx *context.Context) error {
	// #nosec
	return os.MkdirAll(ctx.Config.Dist, 0o755)
}
