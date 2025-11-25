// Package bluesky announces to bluesky.social.
package bluesky

import (
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/lex/util"
	butil "github.com/bluesky-social/indigo/util"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/caarlos0/env/v11"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const (
	defaultPDSURL          = "https://bsky.social"
	defaultMessageTemplate = `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}`
)

// Pipe announcer implementation.
type Pipe struct {
	pdsURL string
}

// New bluesky announcer.
func New() Pipe {
	return Pipe{pdsURL: defaultPDSURL}
}

func (Pipe) String() string { return "bluesky" }
func (Pipe) Skip(ctx *context.Context) (bool, error) {
	enable, err := tmpl.New(ctx).Bool(ctx.Config.Announce.Bluesky.Enabled)
	return !enable, err
}

type Config struct {
	Password string `env:"BLUESKY_APP_PASSWORD,notEmpty"`
}

func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Announce.Bluesky.MessageTemplate == "" {
		ctx.Config.Announce.Bluesky.MessageTemplate = defaultMessageTemplate
	}

	return nil
}

func (p Pipe) Announce(ctx *context.Context) error {
	msg, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Bluesky.MessageTemplate)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	var cfg Config
	if err = env.Parse(&cfg); err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	post := bsky.FeedPost{
		CreatedAt: time.Now().Format(time.RFC3339),
		Text:      msg,
	}

	// if there is a reference to the release URL, create a link to it
	startIdx := strings.Index(msg, ctx.ReleaseURL)
	if startIdx >= 0 {
		post.Facets = []*bsky.RichtextFacet{
			{
				Index: &bsky.RichtextFacet_ByteSlice{
					ByteStart: int64(startIdx),
					ByteEnd:   int64(startIdx + len(ctx.ReleaseURL)),
				},
				Features: []*bsky.RichtextFacet_Features_Elem{
					{
						RichtextFacet_Link: &bsky.RichtextFacet_Link{
							Uri: ctx.ReleaseURL,
						},
					},
				},
			},
		}
	}

	httpClient := butil.RobustHTTPClient()
	userAgent := "goreleaser/v2"

	xrpcClient := &xrpc.Client{
		Client:    httpClient,
		Host:      p.pdsURL,
		UserAgent: &userAgent,
	}

	loginInput := &atproto.ServerCreateSession_Input{
		Identifier: ctx.Config.Announce.Bluesky.Username,
		Password:   cfg.Password,
	}

	authResult, err := atproto.ServerCreateSession(ctx, xrpcClient, loginInput)
	if err != nil {
		return fmt.Errorf("could not log in to Bluesky: %w", err)
	}

	xrpcClient.Auth = &xrpc.AuthInfo{
		AccessJwt:  authResult.AccessJwt,
		RefreshJwt: authResult.RefreshJwt,
		Handle:     authResult.Handle,
		Did:        authResult.Did,
	}

	_, err = atproto.RepoCreateRecord(ctx, xrpcClient, &atproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.post",
		Repo:       xrpcClient.Auth.Did,
		Record: &util.LexiconTypeDecoder{
			Val: &post,
		},
	})

	return err
}
