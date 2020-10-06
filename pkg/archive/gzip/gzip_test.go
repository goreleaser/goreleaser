package gzip

import (
	"compress/gzip"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGzFile(t *testing.T) {
	var assert = require.New(t)
	tmp, err := ioutil.TempDir("", "")
	require.NoError(err)
	f, err := os.Create(filepath.Join(tmp, "test.gz"))
	require.NoError(err)
	defer f.Close() // nolint: errcheck
	archive := New(f)

	require.NoError(archive.Add("sub1/sub2/subfoo.txt", "../testdata/sub1/sub2/subfoo.txt"))
	require.EqualError(archive.Add("foo.txt", "../testdata/foo.txt"), "gzip: failed to add foo.txt, only one file can be archived in gz format")
	require.NoError(archive.Close())

	require.NoError(f.Close())

	t.Log(f.Name())
	f, err = os.Open(f.Name())
	require.NoError(err)
	defer f.Close() // nolint: errcheck

	info, err := f.Stat()
	require.NoError(err)
	require.Truef(info.Size() < 500, "archived file should be smaller than %d", info.Size())

	gzf, err := gzip.NewReader(f)
	require.NoError(err)
	defer gzf.Close() // nolint: errcheck

	require.Equal("sub1/sub2/subfoo.txt", gzf.Name)

	bts, err := ioutil.ReadAll(gzf)
	require.NoError(err)
	require.Equal("sub\n", string(bts))
}
