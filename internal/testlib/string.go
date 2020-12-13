package testlib

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// EqualIgnoreCRLF requires equality ignoring line feed differences.
func EqualIgnoreCRLF(t *testing.T, expected, actual string) {
	require.Equal(t, strings.ReplaceAll(expected, "\r\n", "\n"), strings.ReplaceAll(actual, "\r\n", "\n"))
}
