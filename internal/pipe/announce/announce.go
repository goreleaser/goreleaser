package announce

import (
	"fmt"
	"github.com/apex/log"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe that announces releases on social media platforms
type Pipe struct{}

func (p Pipe) Run(ctx *context.Context) error {
	if !ctx.Config.Twitter.Announce {
		return pipe.ErrSkipPublishEnabled
	}

	config := oauth1.NewConfig(ctx.Config.Twitter.ConsumerKey, ctx.Config.Twitter.ConsumerSecret)
	token := oauth1.NewToken(ctx.Config.Twitter.AccessToken, ctx.Config.Twitter.AccessSecret)
	// http.Client will automatically authorize Requests
	httpClient := config.Client(oauth1.NoContext, token)

	// Twitter client
	client := twitter.NewClient(httpClient)

	msg := ctx.Config.Twitter.MessageTemplate

	parsedMsg, err := tmpl.New(ctx).
		Apply(msg)

	if err != nil {
		fmt.Println(err)
		return err
	}

	log.WithField("message", parsedMsg).Info("sending tweet")

	// Send a Tweet
	_, _, err = client.Statuses.Update(parsedMsg, nil)

	if err != nil {
		log.WithError(err)
		return err
	}
	return nil
}

func (Pipe) String() string {
	return "twitter announcement"
}

// Announcer should be implemented by pipes that want to announce a release.
type Announcer interface {
	fmt.Stringer

	Announce(ctx *context.Context) error
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	return nil
}
