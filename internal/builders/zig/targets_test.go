package zig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckTarget(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		require.Equal(t, targetInvalid, checkTarget("fake-target"))
	})
	t.Run("broken", func(t *testing.T) {
		require.Equal(t, targetBroken, checkTarget("arm-windows-gnu"))
		require.Equal(t, targetBroken, checkTarget("arm-windows"))
	})
	t.Run("valid", func(t *testing.T) {
		require.Equal(t, targetValid, checkTarget("aarch64-linux-musl"))
		require.Equal(t, targetValid, checkTarget("aarch64-linux"))
	})
}
