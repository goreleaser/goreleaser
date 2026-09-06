package publish

import (
	"fmt"
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/dockerdigest"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestPublish(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Release:      config.Release{Disable: "true"},
		DockerDigest: config.DockerDigest{Disable: "true"},
	}, testctx.GitHubTokenType)

	require.NoError(t, New().Run(ctx))
}

func TestPublishPipelineWritesDockerDigestsAfterKo(t *testing.T) {
	pipeline := New().pipeline
	require.Less(t, publisherIndex(t, pipeline, "ko"), publisherIndex(t, pipeline, "docker digests"))
	require.Less(t, publisherIndex(t, pipeline, "docker digests"), publisherIndex(t, pipeline, "signing docker images"))

	ctx := testctx.Wrap(t.Context())
	ctx.Config.Dist = t.TempDir()
	ctx.Config.DockerDigest.NameTemplate = "digests.txt"
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "example.com/app:v1",
		Type: artifact.DockerImage,
		Extra: artifact.Extras{
			artifact.ExtraDigest: "sha256:dockerdigest",
		},
	})

	err := Pipe{
		pipeline: []Publisher{
			&testArtifactPublisher{
				artifact: &artifact.Artifact{
					Name: "example.com/ko:v1",
					Type: artifact.DockerManifest,
					Extra: artifact.Extras{
						artifact.ExtraDigest: "sha256:kodigest",
					},
				},
			},
			dockerdigest.Pipe{},
		},
	}.Run(ctx)
	require.NoError(t, err)

	bts, err := os.ReadFile(ctx.Config.Dist + "/digests.txt")
	require.NoError(t, err)
	require.Equal(t, `dockerdigest  example.com/app:v1
kodigest  example.com/ko:v1
`, string(bts))
}

func TestPublishSuccess(t *testing.T) {
	ctx := testctx.Wrap(t.Context())
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
	require.Equal(t, 1, merr.Len())
	require.True(t, lastStep.ran)
}

func TestPublishError(t *testing.T) {
	ctx := testctx.Wrap(t.Context())
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
		ctx := testctx.Wrap(t.Context(), testctx.Skip(skips.Publish))
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		require.False(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
	})
}

func publisherIndex(tb testing.TB, pipeline []Publisher, name string) int {
	tb.Helper()
	for i, publisher := range pipeline {
		if publisher.String() == name {
			return i
		}
	}
	tb.Fatalf("publisher %q not found", name)
	return -1
}

type testArtifactPublisher struct {
	artifact *artifact.Artifact
}

func (t *testArtifactPublisher) String() string { return "test artifact publisher" }
func (t *testArtifactPublisher) Publish(ctx *context.Context) error {
	ctx.Artifacts.Add(t.artifact)
	return nil
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
