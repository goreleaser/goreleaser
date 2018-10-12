package publish

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestPublishDisable(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.SkipPublish = true
	require.EqualError(t, Pipe{}.Run(ctx), pipe.ErrSkipPublishEnabled.Error())
}

func TestPublish(t *testing.T) {
	var ctx = context.New(config.Project{})
	require.NoError(t, Pipe{}.Run(ctx))
}
