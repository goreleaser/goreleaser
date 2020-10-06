package archive

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArchive(t *testing.T) {
	var assert = require.New(t)
	folder, err := ioutil.TempDir("", "archivetest")
	require.NoError(err)
	empty, err := os.Create(folder + "/empty.txt")
	require.NoError(err)
	require.NoError(os.Mkdir(folder+"/folder-inside", 0755))

	for _, format := range []string{"tar.gz", "zip", "gz", "tar.xz", "willbeatargzanyway"} {
		format := format
		t.Run(format, func(t *testing.T) {
			var archive = newArchive(folder, format, t)
			require.NoError(archive.Add("empty.txt", empty.Name()))
			require.Error(archive.Add("dont.txt", empty.Name()+"_nope"))
			require.NoError(archive.Close())
		})
	}
}

func newArchive(folder, format string, t *testing.T) Archive {
	file, err := os.Create(folder + "/folder." + format)
	require.NoError(t, err)
	return New(file)
}
