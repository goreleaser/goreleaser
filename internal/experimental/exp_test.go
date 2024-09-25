package experimental

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultGoarm(t *testing.T) {
	t.Run("not set", func(t *testing.T) {
		require.Equal(t, "6", DefaultGOARM())
	})
	t.Run("empty", func(t *testing.T) {
		t.Setenv(envKey, "")
		require.Equal(t, "6", DefaultGOARM())
	})
	t.Run("notset", func(t *testing.T) {
		t.Setenv(envKey, "otherexp")
		require.Equal(t, "6", DefaultGOARM())
	})
	t.Run("set", func(t *testing.T) {
		t.Setenv(envKey, "foo,"+defaultGOARMv7+",somethingelse")
		require.Equal(t, "7", DefaultGOARM())
	})
}
