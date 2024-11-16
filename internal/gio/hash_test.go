package gio

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/stretchr/testify/require"
)

func TestEqualFilesError(t *testing.T) {
	tests := []struct {
		a string
		b string
	}{
		{"./testdata/nope.txt", "./testdata/somefile.txt"},
		{"./testdata/somefile.txt", "./testdata/nope.txt"},
	}
	for _, test := range tests {
		equal, err := EqualFiles(test.a, test.b)
		require.Error(t, err)
		require.False(t, equal)

		equalContents, err := EqualFileContents(test.a, test.b)
		require.Error(t, err)
		require.False(t, equalContents)
	}
}

func TestEqualFiles(t *testing.T) {
	tests := []struct {
		a string
		b string
	}{
		{"./testdata/somefile.txt", "./testdata/somefile_copy.txt"},
	}
	for _, test := range tests {
		equal, err := EqualFiles(test.a, test.b)
		require.NoError(t, err)
		require.True(t, equal)

		equalContents, err := EqualFileContents(test.a, test.b)
		require.NoError(t, err)
		require.True(t, equalContents)
	}
}

func TestEqualFileContents(t *testing.T) {
	tests := []struct {
		a string
		b string
	}{
		{"./testdata/somefile.txt", "./testdata/somefile_copy_perm.txt"},
	}
	for _, test := range tests {
		equal, err := EqualFiles(test.a, test.b)
		require.NoError(t, err)
		if !testlib.IsWindows() {
			// this fails on windows due to perms being ignored
			require.False(t, equal)
		}

		equalContents, err := EqualFileContents(test.a, test.b)
		require.NoError(t, err)
		require.True(t, equalContents)
	}
}
