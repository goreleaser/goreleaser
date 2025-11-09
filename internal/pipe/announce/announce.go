// Package announce contains the announcing pipe.
package announce

import (
	"fmt"

	"github.com/goreleaser/goreleaser/v2/internal/middleware/errhandler"
	"github.com/goreleaser/goreleaser/v2/internal/middleware/logging"
	"github.com/goreleaser/goreleaser/v2/internal/middleware/skip"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/bluesky"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/discord"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/discourse"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/linkedin"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/mastodon"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/mattermost"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/opencollective"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/reddit"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/slack"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/smtp"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/teams"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/telegram"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/twitter"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/webhook"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Announcer should be implemented by pipes that want to announce releases.
type Announcer interface {
	fmt.Stringer
	Announce(ctx *context.Context) error
}

//nolint:gochecknoglobals
var announcers = []Announcer{
	// XXX: keep asc sorting
	bluesky.New(),
	discord.Pipe{},
	discourse.Pipe{},
	linkedin.Pipe{},
	mastodon.Pipe{},
	mattermost.Pipe{},
	opencollective.Pipe{},
	reddit.Pipe{},
	slack.Pipe{},
	smtp.Pipe{},
	teams.Pipe{},
	telegram.Pipe{},
	twitter.Pipe{},
	webhook.Pipe{},
}

// Pipe that announces releases.
type Pipe struct{}

func (Pipe) String() string { return "announcing" }

func (Pipe) Skip(ctx *context.Context) (bool, error) {
	if skips.Any(ctx, skips.Announce) {
		return true, nil
	}
	return tmpl.New(ctx).Bool(ctx.Config.Announce.Skip)
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	memo := errhandler.Memo{}
	for _, announcer := range announcers {
		_ = skip.Maybe(
			announcer,
			logging.PadLog(announcer.String(), memo.Wrap(announcer.Announce)),
		)(ctx)
	}
	if memo.Error() != nil {
		return fmt.Errorf("failed to announce release: %w", memo.Error())
	}
	return nil
}
