//go:build integration

package nix

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/golden"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestIntegrationSkipNixAllGood(t *testing.T) {
	testlib.CheckPath(t, "nix-hash")
	testlib.SkipIfWindows(t, "nix doesn't work on windows")
	require.False(t, New().Skip(testctx.WrapWithCfg(t.Context(), config.Project{
		Nix: []config.Nix{{}},
	})))
}

func TestIntegrationHasherHashValid(t *testing.T) {
	testlib.CheckPath(t, "nix-hash")
	testlib.SkipIfWindows(t, "nix doesn't work on windows")
	sha, err := realHasher.Hash(t.Context(), "./testdata/file.bin")
	require.NoError(t, err)
	require.Equal(t, "1n7yy95h81rziah4ppi64kr6fphwxjiq8cl70fpfrqvr0ml1xbcl", sha)
}

func TestIntegrationHasherAvailableValid(t *testing.T) {
	testlib.CheckPath(t, "nix-hash")
	testlib.SkipIfWindows(t, "nix doesn't work on windows")
	require.True(t, realHasher.Available())
}

func TestFormat(t *testing.T) {
	testlib.SkipIfWindows(t, "nix.format won't work on Windows")

	t.Run("invalid formatter", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		require.False(t, format(ctx, "invalid-formatter", "nope.nix"))
	})

	const input = `{  foo = "bar";
							baz = "qux";	}`

	t.Run("alejandra", func(t *testing.T) {
		testlib.CheckPath(t, "alejandra")

		ctx := testctx.Wrap(t.Context())
		path := filepath.Join(t.TempDir(), "test.nix")
		require.NoError(t, os.WriteFile(path, []byte(input), 0o644))

		require.True(t, format(ctx, "alejandra", path))

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		golden.RequireEqualExt(t, content, ".nix")
	})

	t.Run("nixfmt", func(t *testing.T) {
		testlib.CheckPath(t, "nixfmt")

		ctx := testctx.Wrap(t.Context())
		path := filepath.Join(t.TempDir(), "test.nix")
		require.NoError(t, os.WriteFile(path, []byte(input), 0o644))

		require.True(t, format(ctx, "nixfmt", path))

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		golden.RequireEqualExt(t, content, ".nix")
	})

	t.Run("invalid file", func(t *testing.T) {
		testlib.CheckPath(t, "nixfmt")

		ctx := testctx.Wrap(t.Context())
		path := filepath.Join(t.TempDir(), "test.nix")
		require.NoError(t, os.WriteFile(path, []byte(`{ invalid file`), 0o644))

		require.False(t, format(ctx, "nixfmt", path))

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		golden.RequireEqualExt(t, content, ".nix")
	})
}
