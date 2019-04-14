package archive

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArchive(t *testing.T) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(err)
	empty, err := os.Create(folder + "/empty.txt")
	assert.NoError(err)
	assert.NoError(os.Mkdir(folder+"/folder-inside", 0755))

	for _, format := range []string{"tar.gz", "zip", "gz", "willbeatargzanyway"} {
		format := format
		t.Run(format, func(t *testing.T) {
			var archive = newArchive(folder, format, t)
			assert.NoError(archive.Add("empty.txt", empty.Name()))
			assert.Error(archive.Add("dont.txt", empty.Name()+"_nope"))
			assert.NoError(archive.Close())
		})
	}
}

func newArchive(folder, format string, t *testing.T) Archive {
	file, err := os.Create(folder + "/folder." + format)
	assert.NoError(t, err)
	return New(file)
}
