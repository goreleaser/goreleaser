package bluesky

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/caarlos0/env/v9"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	defaultPDSURL          = "https://bsky.social"
	defaultMessageTemplate = `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}`
)

type Pipe struct{}

func (Pipe) String() string                 { return "bluesky" }
func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Config.Announce.BlueSky.Enabled }

type Config struct {
	Password string `env:"BLUESKY_ACCOUNT_PASSWORD,notEmpty"`
}

func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Announce.BlueSky.MessageTemplate == "" {
		ctx.Config.Announce.BlueSky.MessageTemplate = defaultMessageTemplate
	}

	newURL := defaultPDSURL
	if ctx.Config.Announce.BlueSky.PDSURL != "" {
		_, err := url.Parse(ctx.Config.Announce.BlueSky.PDSURL)
		if err != nil {
			return fmt.Errorf("%q is not a valid BlueSky PDS Url: %w", ctx.Config.Announce.BlueSky.PDSURL, err)
		}

		newURL = ctx.Config.Announce.BlueSky.PDSURL
	}

	ctx.Config.Announce.BlueSky.PDSURL = newURL

	return nil
}

func (p Pipe) Announce(ctx *context.Context) error {
	msg, err := tmpl.New(ctx).Apply(ctx.Config.Announce.BlueSky.MessageTemplate)
	if err != nil {
		return fmt.Errorf("bluesky: %w", err)
	}

	var cfg Config
	if err = env.Parse(&cfg); err != nil {
		return fmt.Errorf("bluesky: %w", err)
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

	httpClient := http.DefaultClient
	if strings.TrimSpace(ctx.Config.Announce.BlueSky.CACerts) != "" || ctx.Config.Announce.BlueSky.SkipTLSVerify {
		certPool, err := x509.SystemCertPool()
		if err != nil {
			log.Infof("could not get system cert pool, starting from scratch: %s", err.Error())
			certPool = x509.NewCertPool()
		}
		if strings.TrimSpace(ctx.Config.Announce.BlueSky.CACerts) != "" {
			certPool.AppendCertsFromPEM([]byte(ctx.Config.Announce.BlueSky.CACerts))
		}

		transport, ok := httpClient.Transport.(*http.Transport)
		if !ok {
			return errors.New("this shouldn't happen ever but it's better than a panic. http.DefaultClient.Transport was not a (*http.Transport)")
		}

		transport = transport.Clone()
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: ctx.Config.Announce.BlueSky.SkipTLSVerify,
				RootCAs:            certPool,
			}
		}
		httpClient.Transport = transport
	}

	userAgent := fmt.Sprintf("goreleaser/%s", ctx.Version)

	xrpcClient := &xrpc.Client{
		Client:    httpClient,
		Host:      ctx.Config.Announce.BlueSky.PDSURL,
		UserAgent: &userAgent,
	}

	loginInput := &atproto.ServerCreateSession_Input{
		Identifier: ctx.Config.Announce.BlueSky.Username,
		Password:   cfg.Password,
	}

	authResult, err := atproto.ServerCreateSession(ctx, xrpcClient, loginInput)
	if err != nil {
		return fmt.Errorf("could not log in to BlueSky: %w", err)
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
