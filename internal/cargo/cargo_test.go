package cargo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCargo_name(t *testing.T) {
	cargo, err := Open("./testdata/name.Cargo.toml")
	require.NoError(t, err)
	require.Equal(t, "some-name", cargo.Package.Name)
}

func TestParseCargo_workspaces(t *testing.T) {
	cargo, err := Open("./testdata/workspaces.Cargo.toml")
	require.NoError(t, err)
	require.Len(t, cargo.Workspace.Members, 2)
}
