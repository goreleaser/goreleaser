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
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(err)
	var file = filepath.Join(folder, "subject")
	assert.NoError(ioutil.WriteFile(file, []byte("lorem ipsum"), 0644))
	sum, err := SHA256(file)
	assert.NoError(err)
	assert.Equal("5e2bf57d3f40c4b6df69daf1936cb766f832374b4fc0259a7cbff06e2f70f269", sum)
}

func TestOpenFailure(t *testing.T) {
	var assert = assert.New(t)
	sum, err := SHA256("/tmp/this-file-wont-exist-I-hope")
	assert.Empty(sum)
	assert.Error(err)
}

func TestFileDoesntExist(t *testing.T) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(err)
	var path = filepath.Join(folder, "subject")
	file, err := os.Create(path)
	assert.NoError(err)
	assert.NoError(file.Close())
	_, err = doCalculate(sha256.New(), file)
	assert.Error(err)
}
