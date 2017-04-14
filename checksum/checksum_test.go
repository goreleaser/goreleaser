package checksum

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChecksums(t *testing.T) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "gorelasertest")
	assert.NoError(err)
	file, err := os.OpenFile(
		filepath.Join(folder, "subject"),
		os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0600,
	)
	assert.NoError(err)
	_, err = file.WriteString("lorem ipsum")
	assert.NoError(err)
	assert.NoError(file.Close())
	t.Run("md5", func(t *testing.T) {
		sum, err := MD5(file.Name())
		assert.NoError(err)
		assert.Equal("80a751fde577028640c419000e33eba6", sum)
	})
	t.Run("sha256", func(t *testing.T) {
		sum, err := SHA256(file.Name())
		assert.NoError(err)
		assert.Equal("5e2bf57d3f40c4b6df69daf1936cb766f832374b4fc0259a7cbff06e2f70f269", sum)
	})
}

func TestOpenFailure(t *testing.T) {
	var assert = assert.New(t)
	sum, err := MD5("/tmp/this-file-wont-exist-I-hope")
	assert.Empty(sum)
	assert.Error(err)
}
