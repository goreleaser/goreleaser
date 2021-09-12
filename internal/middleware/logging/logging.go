package logging

import (
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/fatih/color"
	"github.com/goreleaser/goreleaser/internal/middleware"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Padding is a logging initial padding.
type Padding int

// DefaultInitialPadding is the default padding in the log library.
const DefaultInitialPadding Padding = 3

// ExtraPadding is the double of the DefaultInitialPadding.
const ExtraPadding = DefaultInitialPadding * 2

// Log pretty prints the given action and its title.
// You can have different padding levels by providing different initial
// paddings. The middleware will print the title in the given padding and the
// action logs in padding+default padding.
// The default padding in the log library is 3.
// The middleware always resets to the default padding.
func Log(title string, next middleware.Action, padding Padding) middleware.Action {
	return func(ctx *context.Context) error {
		defer func() {
			cli.Default.Padding = int(DefaultInitialPadding)
		}()
		cli.Default.Padding = int(padding)
		log.Infof(color.New(color.Bold).Sprint(title))
		cli.Default.Padding = int(padding + DefaultInitialPadding)
		return next(ctx)
	}
}
