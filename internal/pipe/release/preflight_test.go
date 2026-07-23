package release

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestPreflightDescription(t *testing.T) {
	require.NotEmpty(t, Preflight{}.String())
}

func TestPreflightSkip(t *testing.T) {
	t.Run("skip publish", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.Skip(skips.Publish))
		skip, err := Preflight{}.Skip(ctx)
		require.NoError(t, err)
		require.True(t, skip)
	})

	t.Run("release disabled", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Release: config.Release{Disable: "true"},
		})
		skip, err := Preflight{}.Skip(ctx)
		require.NoError(t, err)
		require.True(t, skip)
	})

	t.Run("release disabled template", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Env: []string{"DISABLE=true"},
			Release: config.Release{
				Disable: "{{ .Env.DISABLE }}",
			},
		})
		skip, err := Preflight{}.Skip(ctx)
		require.NoError(t, err)
		require.True(t, skip)
	})

	t.Run("do not skip normal release", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.GitHubTokenType)
		skip, err := Preflight{}.Skip(ctx)
		require.NoError(t, err)
		require.False(t, skip)
	})

	t.Run("skip token check (goreleaser build)", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.GitHubTokenType)
		ctx.SkipTokenCheck = true
		skip, err := Preflight{}.Skip(ctx)
		require.NoError(t, err)
		require.True(t, skip)
	})

	t.Run("bad template", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Release: config.Release{Disable: "{{ .Env.MISSING }}"},
		})
		_, err := Preflight{}.Skip(ctx)
		require.Error(t, err)
	})
}

func TestPreflightRun(t *testing.T) {
	t.Run("check passes", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		mock := client.NewMock()
		require.NoError(t, runPreflight(ctx, mock))
		require.True(t, mock.CanReleaseCalled)
	})

	t.Run("check fails, warn by default", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		mock := &client.Mock{FailCanRelease: true}
		// Without fail_on_error the failing check must not abort: it only warns.
		require.NoError(t, runPreflight(ctx, mock))
		require.True(t, mock.CanReleaseCalled)
	})

	t.Run("check fails, fail_on_error aborts", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Release: config.Release{
				Preflight: config.ReleasePreflight{FailOnError: "true"},
			},
		})
		mock := &client.Mock{FailCanRelease: true}
		require.Error(t, runPreflight(ctx, mock))
		require.True(t, mock.CanReleaseCalled)
	})

	t.Run("check fails, fail_on_error templated", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Env: []string{"FAIL=true"},
			Release: config.Release{
				Preflight: config.ReleasePreflight{FailOnError: "{{ .Env.FAIL }}"},
			},
		})
		require.Error(t, runPreflight(ctx, &client.Mock{FailCanRelease: true}))
	})

	t.Run("check fails, fail_on_error bad template", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Release: config.Release{
				Preflight: config.ReleasePreflight{FailOnError: "{{ .Env.MISSING }}"},
			},
		})
		require.Error(t, runPreflight(ctx, &client.Mock{FailCanRelease: true}))
	})

	t.Run("client does not implement ReleaseChecker", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		require.NoError(t, runPreflight(ctx, &noopClient{}))
	})
}

// noopClient is a minimal client that does NOT implement ReleaseChecker.
type noopClient struct{}

func (noopClient) CloseMilestone(_ *context.Context, _ client.Repo, _ string) error {
	return nil
}
func (noopClient) CreateRelease(_ *context.Context, _ string) (string, error) { return "", nil }
func (noopClient) PublishRelease(_ *context.Context, _ string) error          { return nil }
func (noopClient) Upload(_ *context.Context, _ string, _ *artifact.Artifact) error {
	return nil
}

func (noopClient) Changelog(_ *context.Context, _ client.Repo, _, _ string) ([]client.ChangelogItem, error) {
	return nil, nil
}
func (noopClient) ReleaseURLTemplate(_ *context.Context) (string, error) { return "", nil }
func (noopClient) CreateFile(_ *context.Context, _ config.CommitAuthor, _ client.Repo, _ []byte, _, _ string) error {
	return nil
}

// Make noopClient satisfy the compiler.
var _ client.Client = noopClient{}
