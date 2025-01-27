package testlib

import (
	"slices"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/stretchr/testify/require"
)

func RequireEqualArtifacts(tb testing.TB, expected, got []*artifact.Artifact) {
	tb.Helper()
	slices.SortFunc(expected, artifactSort)
	slices.SortFunc(got, artifactSort)
	require.Equal(tb, filenames(expected), filenames(got))
	for i := range expected {
		require.Equal(tb, *expected[i], *got[i], "item %d", i)
	}
	require.Equal(tb, expected, got)
}

func artifactSort(a, b *artifact.Artifact) int {
	return strings.Compare(a.Path, b.Path)
}

func filenames(ts []*artifact.Artifact) []string {
	result := make([]string, len(ts))
	for i, t := range ts {
		result[i] = t.Path
	}
	return result
}
