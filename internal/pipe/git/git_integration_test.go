//go:build integration

package git

import (
	"os/exec"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/stretchr/testify/require"
)

func TestIntegrationShallowClone(t *testing.T) {
	folder := testlib.Mktmp(t)
	require.NoError(
		t,
		exec.CommandContext(
			t.Context(),
			"git", "clone",
			"--depth", "1",
			"--branch", "v0.160.0",
			"https://github.com/goreleaser/goreleaser",
			folder,
		).Run(),
	)
	t.Run("all checks up", func(t *testing.T) {
		// its just a warning now
		require.NoError(t, Pipe{}.Run(testctx.Wrap(t.Context())))
	})
	t.Run("skip validate is set", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.Skip(skips.Validate))
		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	})
	t.Run("snapshot", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.Snapshot)
		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	})
}
