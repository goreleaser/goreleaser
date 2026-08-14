package gentoo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetermineKeepLatestDeletions(t *testing.T) {
	prefix := "foo-"
	t.Run("keeps new ebuild and deletes older existing one", func(t *testing.T) {
		ebuilds := []string{"foo-1.0.0.ebuild"}
		newFiles := []string{"foo-1.1.0.ebuild"}
		toDelete := determineKeepLatestDeletions(ebuilds, newFiles, prefix, 1)
		require.Equal(t, []string{"foo-1.0.0.ebuild"}, toDelete)
	})

	t.Run("keeps newer existing ebuild and deletes older new one", func(t *testing.T) {
		ebuilds := []string{"foo-1.1.0.ebuild"}
		newFiles := []string{"foo-1.0.0.ebuild"}
		toDelete := determineKeepLatestDeletions(ebuilds, newFiles, prefix, 1)
		require.Empty(t, toDelete)
	})

	t.Run("keeps latest N ebuilds correctly sorted", func(t *testing.T) {
		ebuilds := []string{"foo-1.0.0.ebuild", "foo-2.0.0.ebuild", "foo-0.9.0.ebuild"}
		newFiles := []string{"foo-1.5.0.ebuild", "foo-3.0.0.ebuild"}
		toDelete := determineKeepLatestDeletions(ebuilds, newFiles, prefix, 3)

		// All ebuilds: 3.0.0, 2.0.0, 1.5.0, 1.0.0, 0.9.0
		// Kept: 3.0.0, 2.0.0, 1.5.0
		// We expect existing 1.0.0 and 0.9.0 to be deleted.
		require.ElementsMatch(t, []string{"foo-1.0.0.ebuild", "foo-0.9.0.ebuild"}, toDelete)
	})

	t.Run("handles invalid version gracefully and retains alphabetical sort", func(t *testing.T) {
		ebuilds := []string{"foo-badversion.ebuild", "foo-1.0.0.ebuild"}
		newFiles := []string{"foo-2.0.0.ebuild"}
		// valid ones: 2.0.0, 1.0.0. badversion is fallback
		toDelete := determineKeepLatestDeletions(ebuilds, newFiles, prefix, 2)
		require.ElementsMatch(t, []string{"foo-badversion.ebuild"}, toDelete)
	})

	t.Run("deletes nothing when within limits", func(t *testing.T) {
		ebuilds := []string{"foo-1.0.0.ebuild"}
		newFiles := []string{"foo-2.0.0.ebuild"}
		toDelete := determineKeepLatestDeletions(ebuilds, newFiles, prefix, 2)
		require.Empty(t, toDelete)
	})
}
