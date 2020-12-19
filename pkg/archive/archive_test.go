package archive

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArchive(t *testing.T) {
	var folder = t.TempDir()
	empty, err := os.Create(folder + "/empty.txt")
	require.NoError(t, err)
	require.NoError(t, os.Mkdir(folder+"/folder-inside", 0755))

	for _, format := range []string{"tar.gz", "zip", "gz", "tar.xz", "willbeatargzanyway"} {
		format := format
		t.Run(format, func(t *testing.T) {
			var archive = newArchive(folder, format, t)
			require.NoError(t, archive.Add("empty.txt", empty.Name()))
			require.Error(t, archive.Add("dont.txt", empty.Name()+"_nope"))
			require.NoError(t, archive.Close())
		})
	}
}

func newArchive(folder, format string, t *testing.T) Archive {
	file, err := os.Create(folder + "/folder." + format)
	require.NoError(t, err)
	return New(file)
}
