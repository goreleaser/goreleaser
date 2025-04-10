package testlib

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/stretchr/testify/require"
)

func RequireNoExtraField(tb testing.TB, a *artifact.Artifact, key string) {
	tb.Helper()
	_, ok := a.Extra[key]
	require.False(tb, ok)
}

func RequireEqualArtifacts(tb testing.TB, expected, got []*artifact.Artifact) {
	tb.Helper()
	slices.SortFunc(expected, artifactSort)
	slices.SortFunc(got, artifactSort)
	require.Equal(tb, filenames(expected), filenames(got))
	for i := range expected {
		a, b := *expected[i], *got[i]
		require.ElementsMatch(
			tb,
			slices.Collect(maps.Keys(a.Extra)),
			slices.Collect(maps.Keys(b.Extra)),
			"extra keys don't match",
		)
		for k, v := range a.Extra {
			if reflect.TypeOf(v).Kind() == reflect.Slice {
				require.ElementsMatch(
					tb,
					v,
					b.Extra[k],
					"extra values don't match",
				)
				continue
			}
			require.Equal(
				tb,
				a.Extra[k],
				b.Extra[k],
				"extra values don't match",
			)
		}

		// Delete the extra map to avoid running into order errors.
		a.Extra = nil
		b.Extra = nil
		require.Equal(tb, a, b, "elements don't match")
	}
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
