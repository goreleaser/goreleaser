package announce

import (
	"errors"
	"testing"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestAnnounce(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			Twitter: config.Twitter{
				Enabled: true,
			},
			Mastodon: config.Mastodon{
				Enabled: true,
				Server:  "https://localhost:1234/",
			},
		},
	})
	err := Pipe{}.Run(ctx)
	require.Error(t, err)
	merr := &multierror.Error{}
	require.True(t, errors.As(err, &merr), "must be a multierror")
	require.Len(t, merr.Errors, 2)
}

func TestAnnounceAllDisabled(t *testing.T) {
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Run(ctx))
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		ctx := testctx.New(testctx.SkipAnnounce)
		b, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.True(t, b)
	})

	t.Run("skip on patches", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				Skip: "{{gt .Patch 0}}",
			},
		}, testctx.WithSemver(0, 0, 1, ""))
		b, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.True(t, b)
	})

	t.Run("invalid template", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				Skip: "{{if eq .Patch 123}",
			},
		}, testctx.WithSemver(0, 0, 1, ""))
		_, err := Pipe{}.Skip(ctx)
		require.Error(t, err)
	})

	t.Run("dont skip", func(t *testing.T) {
		b, err := Pipe{}.Skip(testctx.New())
		require.NoError(t, err)
		require.False(t, b)
	})

	t.Run("dont skip based on template", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				Skip: "{{gt .Patch 0}}",
			},
		})
		b, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.False(t, b)
	})
}
