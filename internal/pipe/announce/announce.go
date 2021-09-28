// Package announce contains the announcing pipe.
package announce

import (
	"fmt"

	"github.com/goreleaser/goreleaser/internal/middleware/errhandler"
	"github.com/goreleaser/goreleaser/internal/middleware/logging"
	"github.com/goreleaser/goreleaser/internal/middleware/skip"
	"github.com/goreleaser/goreleaser/internal/pipe/discord"
	"github.com/goreleaser/goreleaser/internal/pipe/mattermost"
	"github.com/goreleaser/goreleaser/internal/pipe/reddit"
	"github.com/goreleaser/goreleaser/internal/pipe/slack"
	"github.com/goreleaser/goreleaser/internal/pipe/smtp"
	"github.com/goreleaser/goreleaser/internal/pipe/teams"
	"github.com/goreleaser/goreleaser/internal/pipe/twitter"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Announcer should be implemented by pipes that want to announce releases.
type Announcer interface {
	fmt.Stringer

	Announce(ctx *context.Context) error
}

// nolint: gochecknoglobals
var announcers = []Announcer{
	// XXX: keep asc sorting
	discord.Pipe{},
	mattermost.Pipe{},
	reddit.Pipe{},
	slack.Pipe{},
	smtp.Pipe{},
	teams.Pipe{},
	twitter.Pipe{},
}

// Pipe that announces releases.
type Pipe struct{}

func (Pipe) String() string                 { return "announcing" }
func (Pipe) Skip(ctx *context.Context) bool { return ctx.SkipAnnounce }

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	for _, announcer := range announcers {
		if err := skip.Maybe(
			announcer,
			logging.Log(
				announcer.String(),
				errhandler.Handle(announcer.Announce),
				logging.ExtraPadding,
			),
		)(ctx); err != nil {
			return fmt.Errorf("%s: failed to announce release: %w", announcer.String(), err)
		}
	}
	return nil
}
