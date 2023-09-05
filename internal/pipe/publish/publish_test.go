package publish

import (
	"fmt"
	"testing"

	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestPublish(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Release: config.Release{Disable: "true"},
	}, testctx.GitHubTokenType)
	require.NoError(t, New().Run(ctx))
}

func TestPublishSuccess(t *testing.T) {
	ctx := testctx.New()
	lastStep := &testPublisher{}
	err := Pipe{
		pipeline: []Publisher{
			&testPublisher{},
			&testPublisher{shouldSkip: true},
			&testPublisher{
				shouldErr:   true,
				continuable: true,
			},
			&testPublisher{shouldSkip: true},
			&testPublisher{},
			&testPublisher{shouldSkip: true},
			lastStep,
		},
	}.Run(ctx)
	require.Error(t, err)
	merr := &multierror.Error{}
	require.ErrorAs(t, err, &merr)
	require.Equal(t, merr.Len(), 1)
	require.True(t, lastStep.ran)
}

func TestPublishError(t *testing.T) {
	ctx := testctx.New()
	lastStep := &testPublisher{}
	err := Pipe{
		pipeline: []Publisher{
			&testPublisher{},
			&testPublisher{shouldSkip: true},
			&testPublisher{
				shouldErr:   true,
				continuable: true,
			},
			&testPublisher{},
			&testPublisher{shouldSkip: true},
			&testPublisher{},
			&testPublisher{shouldErr: true},
			lastStep,
		},
	}.Run(ctx)
	require.Error(t, err)
	require.EqualError(t, err, "test: failed to publish artifacts: errored")
	require.False(t, lastStep.ran)
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		ctx := testctx.New(testctx.Skip(skips.Publish))
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		require.False(t, Pipe{}.Skip(testctx.New()))
	})
}

type testPublisher struct {
	shouldErr   bool
	shouldSkip  bool
	continuable bool
	ran         bool
}

func (t *testPublisher) ContinueOnError() bool { return t.continuable }
func (t *testPublisher) String() string        { return "test" }
func (t *testPublisher) Publish(_ *context.Context) error {
	if t.shouldSkip {
		return pipe.Skip("skipped")
	}
	if t.shouldErr {
		return fmt.Errorf("errored")
	}
	t.ran = true
	return nil
}
