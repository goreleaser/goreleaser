package ids

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIDs(t *testing.T) {
	var ids = New("foos")
	ids.Inc("foo")
	ids.Inc("bar")
	require.NoError(t, ids.Validate())
}

func TestIDsError(t *testing.T) {
	var ids = New("builds")
	ids.Inc("foo")
	ids.Inc("bar")
	ids.Inc("foo")
	require.EqualError(t, ids.Validate(), "found 2 builds with the ID 'foo', please fix your config")
}
