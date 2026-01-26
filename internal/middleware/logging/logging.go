// Package logging contains logging middleware.
package logging

import (
	"time"

	"charm.land/lipgloss/v2"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/internal/middleware"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

var (
	bold      = lipgloss.NewStyle().Bold(true)
	threshold = time.Second * 10
)

// Log pretty prints the given action and its title.
func Log(title string, next middleware.Action) middleware.Action {
	return func(ctx *context.Context) error {
		start := time.Now()
		defer func() {
			logext.Duration(start, threshold)
			log.ResetPadding()
		}()
		if title != "" {
			log.Infof(bold.Render(title))
			log.IncreasePadding()
		}
		return next(ctx)
	}
}

// PadLog pretty prints the given action and its title with an increased padding.
func PadLog(title string, next middleware.Action) middleware.Action {
	return func(ctx *context.Context) error {
		start := time.Now()
		defer func() {
			logext.Duration(start, threshold)
			log.ResetPadding()
		}()
		log.ResetPadding()
		log.IncreasePadding()
		log.Infof(bold.Render(title))
		log.IncreasePadding()
		return next(ctx)
	}
}
