package custompublishers

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			Publishers: []config.Publisher{
				{},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestPublish(t *testing.T) {
	require.NoError(t, Pipe{}.Publish(context.New(config.Project{
		Publishers: []config.Publisher{
			{
				Cmd: "echo",
			},
		},
	})))
}
