package logging

import (
	"github.com/caarlos0/log"
	"github.com/charmbracelet/lipgloss"
	"github.com/goreleaser/goreleaser/internal/middleware"
	"github.com/goreleaser/goreleaser/pkg/context"
)

var bold = lipgloss.NewStyle().Bold(true)

// Log pretty prints the given action and its title.
func Log(title string, next middleware.Action) middleware.Action {
	return func(ctx *context.Context) error {
		defer func() {
			log.ResetPadding()
		}()
		log.Infof(bold.Render(title))
		log.IncreasePadding()
		return next(ctx)
	}
}

// PadLog pretty prints the given action and its title with an increased padding.
func PadLog(title string, next middleware.Action) middleware.Action {
	return func(ctx *context.Context) error {
		defer log.ResetPadding()
		log.IncreasePadding()
		log.Infof(bold.Render(title))
		log.IncreasePadding()
		return next(ctx)
	}
}
