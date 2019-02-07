package gzip

import (
	"compress/gzip"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGzFile(t *testing.T) {
	var assert = assert.New(t)
	tmp, err := ioutil.TempDir("", "")
	assert.NoError(err)
	f, err := os.Create(filepath.Join(tmp, "test.gz"))
	assert.NoError(err)
	defer f.Close() // nolint: errcheck
	archive := New(f)

	assert.NoError(archive.Add("sub1/sub2/subfoo.txt", "../testdata/sub1/sub2/subfoo.txt"))
	assert.EqualError(archive.Add("foo.txt", "../testdata/foo.txt"), "gzip: failed to add foo.txt, only one file can be archived in gz format")
	assert.NoError(archive.Close())

	assert.NoError(f.Close())

	t.Log(f.Name())
	f, err = os.Open(f.Name())
	assert.NoError(err)
	defer f.Close() // nolint: errcheck

	info, err := f.Stat()
	assert.NoError(err)
	assert.Truef(info.Size() < 500, "archived file should be smaller than %d", info.Size())

	gzf, err := gzip.NewReader(f)
	assert.NoError(err)
	defer gzf.Close() // nolint: errcheck

	assert.Equal("sub1/sub2/subfoo.txt", gzf.Name)

	bts, err := ioutil.ReadAll(gzf)
	assert.NoError(err)
	assert.Equal("sub\n", string(bts))
}
