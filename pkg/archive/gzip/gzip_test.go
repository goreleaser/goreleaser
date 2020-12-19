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
	var tmp = t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "test.gz"))
	require.NoError(t, err)
	defer f.Close() // nolint: errcheck
	archive := New(f)

	require.NoError(t, archive.Add("sub1/sub2/subfoo.txt", "../testdata/sub1/sub2/subfoo.txt"))
	require.EqualError(t, archive.Add("foo.txt", "../testdata/foo.txt"), "gzip: failed to add foo.txt, only one file can be archived in gz format")
	require.NoError(t, archive.Close())

	require.NoError(t, f.Close())

	t.Log(f.Name())
	f, err = os.Open(f.Name())
	require.NoError(t, err)
	defer f.Close() // nolint: errcheck

	info, err := f.Stat()
	require.NoError(t, err)
	require.Truef(t, info.Size() < 500, "archived file should be smaller than %d", info.Size())

	gzf, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gzf.Close() // nolint: errcheck

	require.Equal(t, "sub1/sub2/subfoo.txt", gzf.Name)

	bts, err := ioutil.ReadAll(gzf)
	require.NoError(t, err)
	require.Equal(t, "sub\n", string(bts))
}
