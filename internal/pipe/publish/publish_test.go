package publish

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestPublish(t *testing.T) {
	ctx := context.New(config.Project{})
	ctx.Config.Release.Disable = true
	ctx.TokenType = context.TokenTypeGitHub
	for i := range ctx.Config.Dockers {
		ctx.Config.Dockers[i].SkipPush = "true"
	}
	require.NoError(t, Pipe{}.Run(ctx))
}
