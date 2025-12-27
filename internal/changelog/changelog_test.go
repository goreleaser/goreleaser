package changelog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractCoAuthors(t *testing.T) {
	authors := ExtractCoAuthors(`foo lala

Co-authored-by: Name <name@example.com>
co-authored-by: Another Name <another-name@example.com>
Assisted-by: Crush <charm@lalla>

`)
	require.Len(t, authors, 2)
	require.Equal(t, []Author{
		{
			Name:  "Name",
			Email: "name@example.com",
		},
		{
			Name:  "Another Name",
			Email: "another-name@example.com",
		},
	}, authors)
}
