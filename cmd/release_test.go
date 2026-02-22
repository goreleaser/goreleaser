package cmd

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestReleaseFlags(t *testing.T) {
	setup := func(tb testing.TB, opts releaseOpts) *context.Context {
		tb.Helper()
		ctx := testctx.Wrap(t.Context())
		require.NoError(t, setupReleaseContext(ctx, opts))
		return ctx
	}

	t.Run("draft", func(t *testing.T) {
		t.Run("not set", func(t *testing.T) {
			ctx := setup(t, releaseOpts{})
			require.False(t, ctx.Config.Release.Draft)
		})

		t.Run("set via flag", func(t *testing.T) {
			ctx := setup(t, releaseOpts{
				draft: true,
			})
			require.True(t, ctx.Config.Release.Draft)
		})

		t.Run("set in config", func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Release: config.Release{
					Draft: true,
				},
			})

			require.NoError(t, setupReleaseContext(ctx, releaseOpts{}))
			require.True(t, ctx.Config.Release.Draft)
		})
	})

	t.Run("action", func(t *testing.T) {
		ctx := setup(t, releaseOpts{})
		require.Equal(t, context.ActionRelease, ctx.Action)
	})

	t.Run("snapshot", func(t *testing.T) {
		ctx := setup(t, releaseOpts{
			snapshot: true,
		})
		require.True(t, ctx.Snapshot)
		requireAll(t, ctx, skips.Publish, skips.Validate, skips.Announce)
	})

	t.Run("skips", func(t *testing.T) {
		ctx := setup(t, releaseOpts{
			skips: []string{
				string(skips.Publish),
				string(skips.Sign),
				string(skips.Validate),
			},
		})

		requireAll(t, ctx, skips.Sign, skips.Publish, skips.Validate, skips.Announce)
	})

	t.Run("parallelism", func(t *testing.T) {
		require.Equal(t, 1, setup(t, releaseOpts{
			parallelism: 1,
		}).Parallelism)
	})

	t.Run("notes", func(t *testing.T) {
		notes := "foo.md"
		header := "header.md"
		footer := "footer.md"
		ctx := setup(t, releaseOpts{
			releaseNotesFile:  notes,
			releaseHeaderFile: header,
			releaseFooterFile: footer,
		})
		require.Equal(t, notes, ctx.ReleaseNotesFile)
		require.Equal(t, header, ctx.ReleaseHeaderFile)
		require.Equal(t, footer, ctx.ReleaseFooterFile)
	})

	t.Run("templated notes", func(t *testing.T) {
		notes := "foo.md"
		header := "header.md"
		footer := "footer.md"
		ctx := setup(t, releaseOpts{
			releaseNotesTmpl:  notes,
			releaseHeaderTmpl: header,
			releaseFooterTmpl: footer,
		})
		require.Equal(t, notes, ctx.ReleaseNotesTmpl)
		require.Equal(t, header, ctx.ReleaseHeaderTmpl)
		require.Equal(t, footer, ctx.ReleaseFooterTmpl)
	})

	t.Run("rm dist", func(t *testing.T) {
		require.True(t, setup(t, releaseOpts{
			clean: true,
		}).Clean)
	})
}
