package build

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestFailBuilder(t *testing.T) {
	f := For("a")
	b, ok := f.(failBuilder)
	require.True(t, ok)
	_, err := b.WithDefaults(config.Build{})
	require.EqualError(t, err, b.err.Error())
	require.EqualError(t, b.Build(nil, config.Build{}, Options{}), b.err.Error())
	_, err = b.Parse("")
	require.EqualError(t, err, b.err.Error())
	require.Empty(t, b.Dependencies())
	require.EqualError(t, b.Prepare(nil, config.Build{}), b.err.Error())
	require.False(t, b.AllowConcurrentBuilds())
	require.Empty(t, b.FixTarget("a"))
}
