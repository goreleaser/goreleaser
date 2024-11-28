package zig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckTarget(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		for _, target := range []string{
			"fake-target",
			"x86_64-windows-abcde",
		} {
			t.Run(target, func(t *testing.T) {
				require.Equal(t, targetInvalid, checkTarget(target))
			})
		}
	})
	t.Run("broken", func(t *testing.T) {
		for _, target := range []string{
			"arm-windows",
			"arm-windows-gnu",
		} {
			t.Run(target, func(t *testing.T) {
				require.Equal(t, targetBroken, checkTarget(target))
			})
		}
	})
	t.Run("valid", func(t *testing.T) {
		for _, target := range []string{
			"aarch64-linux-musl",
			"aarch64-linux",
			"aarch64-macos",
		} {
			t.Run(target, func(t *testing.T) {
				require.Equal(t, targetValid, checkTarget(target))
			})
		}
	})
}
