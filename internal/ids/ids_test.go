package ids

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIDs(t *testing.T) {
	var ids = New()
	ids.Inc("foo")
	ids.Inc("bar")
	require.NoError(t, ids.Validate())
}

func TestIDsError(t *testing.T) {
	var ids = New()
	ids.Inc("foo")
	ids.Inc("bar")
	ids.Inc("foo")
	require.EqualError(t, ids.Validate(), "found 2 items with the ID 'foo', please fix your config")
}
