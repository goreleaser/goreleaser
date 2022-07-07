package client

import "github.com/goreleaser/goreleaser/pkg/config"

func getReleaseNotes(existing, current string, mode config.ReleaseNotesMode) string {
	switch mode {
	case config.ReleaseNotesModeAppend:
		return existing + "\n\n" + current
	case config.ReleaseNotesModeReplace:
		return current
	case config.ReleaseNotesModePrepend:
		return current + "\n\n" + existing
	default:
		if existing != "" {
			return existing
		}
		return current
	}
}
