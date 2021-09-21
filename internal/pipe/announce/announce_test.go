package announce

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
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
		},
	})
	require.Error(t, Pipe{}.Run(ctx))
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

	t.Run("dont skip", func(t *testing.T) {
		require.False(t, Pipe{}.Skip(context.New(config.Project{})))
	})
}
