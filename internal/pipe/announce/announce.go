// Package announce contains the announcing pipe.
package announce

import (
	"fmt"

	"github.com/goreleaser/goreleaser/internal/middleware"
	"github.com/goreleaser/goreleaser/internal/pipe/reddit"
	"github.com/goreleaser/goreleaser/internal/pipe/slack"
	"github.com/goreleaser/goreleaser/internal/pipe/twitter"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe that announces releases.
type Pipe struct{}

func (Pipe) String() string {
	return "announcing"
}

// Announcer should be implemented by pipes that want to announce releases.
type Announcer interface {
	fmt.Stringer

	Announce(ctx *context.Context) error
}

// nolint: gochecknoglobals
var announcers = []Announcer{
	twitter.Pipe{}, // announce to twitter
	reddit.Pipe{},  // announce to twitter
	slack.Pipe{},   // announce to slack
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	for _, announcer := range announcers {
		if err := middleware.Logging(
			announcer.String(),
			middleware.ErrHandler(announcer.Announce),
			middleware.ExtraPadding,
		)(ctx); err != nil {
			return fmt.Errorf("%s: failed to announce release: %w", announcer.String(), err)
		}
	}
	return nil
}
