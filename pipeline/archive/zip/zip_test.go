package zip

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZipFile(t *testing.T) {
	var assert = assert.New(t)

	folder, err := ioutil.TempDir("", "ziptest")
	assert.NoError(err)

	file, err := os.Create(folder + "/folder.zip")
	assert.NoError(err)

	empty, err := os.Create(folder + "/empty.txt")
	assert.NoError(err)

	archive := New(file)
	assert.NoError(archive.Add("empty.txt", empty.Name()))
	assert.Error(archive.Add("dont.txt", empty.Name()+"_nope"))
	assert.NoError(archive.Close())
}
