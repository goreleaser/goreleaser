package packagejson

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	pkg, err := Open("./testdata/package.json")
	require.NoError(t, err)
	require.Equal(t, "example", pkg.Name)
	require.Equal(t, "index.ts", pkg.Module)
	require.Equal(t, "module", pkg.Type)
	require.True(t, pkg.IsBun())
}
