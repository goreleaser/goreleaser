package logging

import (
	"github.com/caarlos0/log"
	"github.com/caarlos0/log/handlers/cli"
	"github.com/fatih/color"
	"github.com/goreleaser/goreleaser/internal/middleware"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Log pretty prints the given action and its title.
func Log(title string, next middleware.Action) middleware.Action {
	return func(ctx *context.Context) error {
		defer func() {
			cli.Default.ResetPadding()
		}()
		log.Infof(color.New(color.Bold).Sprint(title))
		cli.Default.IncreasePadding()
		return next(ctx)
	}
}

// PadLog pretty prints the given action and its title with an increased padding.
func PadLog(title string, next middleware.Action) middleware.Action {
	return func(ctx *context.Context) error {
		defer func() {
			cli.Default.ResetPadding()
		}()
		cli.Default.IncreasePadding()
		log.Infof(color.New(color.Bold).Sprint(title))
		cli.Default.IncreasePadding()
		return next(ctx)
	}
}
