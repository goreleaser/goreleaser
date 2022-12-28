package announce

import (
	"errors"
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestAnnounce(t *testing.T) {
	ctx := context.New(config.Project{
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
	ctx := context.New(config.Project{})
	require.NoError(t, Pipe{}.Run(ctx))
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		ctx := context.New(config.Project{})
		ctx.SkipAnnounce = true
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("skip on patches", func(t *testing.T) {
		ctx := context.New(config.Project{
			Announce: config.Announce{
				Skip: "{{gt .Patch 0}}",
			},
		})
		ctx.Semver.Patch = 1
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("skip on invalid template", func(t *testing.T) {
		ctx := context.New(config.Project{
			Announce: config.Announce{
				Skip: "{{if eq .Patch 123}",
			},
		})
		ctx.Semver.Patch = 1
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		require.False(t, Pipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("dont skip based on template", func(t *testing.T) {
		ctx := context.New(config.Project{
			Announce: config.Announce{
				Skip: "{{gt .Patch 0}}",
			},
		})
		ctx.Semver.Patch = 0
		require.False(t, Pipe{}.Skip(ctx))
	})
}
