package rust

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCargo(t *testing.T) {
	cargo, err := parseCargo("./testdata/workplaces.Cargo.toml")
	require.NoError(t, err)
	require.Len(t, cargo.Workspace.Members, 2)
}
