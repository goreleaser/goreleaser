package checksum

import (
	"crypto/sha256"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChecksums(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	var file = filepath.Join(folder, "subject")
	assert.NoError(t, ioutil.WriteFile(file, []byte("lorem ipsum"), 0644))
	sum, err := SHA256(file)
	assert.NoError(t, err)
	assert.Equal(t, "5e2bf57d3f40c4b6df69daf1936cb766f832374b4fc0259a7cbff06e2f70f269", sum)
}

func TestOpenFailure(t *testing.T) {
	sum, err := SHA256("/tmp/this-file-wont-exist-I-hope")
	assert.Empty(t, sum)
	assert.Error(t, err)
}

func TestFileDoesntExist(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	var path = filepath.Join(folder, "subject")
	file, err := os.Create(path)
	assert.NoError(t, err)
	assert.NoError(t, file.Close())
	_, err = doCalculate(sha256.New(), file)
	assert.Error(t, err)
}
