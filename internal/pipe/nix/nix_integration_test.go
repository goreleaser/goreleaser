//go:build integration

package nix

import (
	"testing"

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
