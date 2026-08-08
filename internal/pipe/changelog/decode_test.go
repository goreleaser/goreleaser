package changelog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecode(t *testing.T) {
	line := func(message, body, author, email string) string {
		return shaOpen + "abc123" + shaClose +
			messageOpen + message + messageClose +
			messageBodyOpen + body + messageBodyClose +
			authorOpen + author + authorClose +
			emailOpen + email + emailClose
	}

	t.Run("well formed", func(t *testing.T) {
		item := decode(line("fix: something", "", "Someone", "someone@example.com"))
		require.Equal(t, "abc123", item.SHA)
		require.Equal(t, "fix: something", item.Message)
		require.Equal(t, "Someone", item.AuthorName)
		require.Equal(t, "someone@example.com", item.AuthorEmail)
	})

	t.Run("subject containing a marker", func(t *testing.T) {
		// git puts the subject in verbatim, so a commit that mentions one of
		// the markers used to move a closing index before its opening one
		item := decode(line("fix: stop emitting </goreleaser_email>", "", "Someone", "someone@example.com"))
		require.Equal(t, "abc123", item.SHA)
		require.Equal(t, "someone@example.com", item.AuthorEmail)
	})

	t.Run("missing markers", func(t *testing.T) {
		item := decode(shaOpen + "abc123" + shaClose + messageOpen + "fix: something" + messageClose)
		require.Equal(t, "abc123", item.SHA)
		require.Equal(t, "fix: something", item.Message)
		require.Empty(t, item.AuthorName)
		require.Empty(t, item.AuthorEmail)
	})
}
