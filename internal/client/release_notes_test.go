package client

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestGetReleaseNotes(t *testing.T) {
	const existing = "existing rel notes"
	const current = "current rel notes"

	t.Run("keep and existing empty", func(t *testing.T) {
		require.Equal(t, current, getReleaseNotes("", current, config.ReleaseNotesModeKeepExisting))
	})

	t.Run("keep", func(t *testing.T) {
		require.Equal(t, existing, getReleaseNotes(existing, current, config.ReleaseNotesModeKeepExisting))
	})

	t.Run("replace", func(t *testing.T) {
		require.Equal(t, current, getReleaseNotes(existing, current, config.ReleaseNotesModeReplace))
	})

	t.Run("append", func(t *testing.T) {
		require.Equal(t, "existing rel notes\n\ncurrent rel notes", getReleaseNotes(existing, current, config.ReleaseNotesModeAppend))
	})

	t.Run("prepend", func(t *testing.T) {
		require.Equal(t, "current rel notes\n\nexisting rel notes", getReleaseNotes(existing, current, config.ReleaseNotesModePrepend))
	})

	t.Run("invalid", func(t *testing.T) {
		require.Equal(t, existing, getReleaseNotes(existing, current, config.ReleaseNotesMode("invalid")))
	})
}
