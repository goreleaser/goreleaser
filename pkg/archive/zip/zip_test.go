package zip

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestZipFile(t *testing.T) {
	var assert = require.New(t)
	tmp, err := ioutil.TempDir("", "")
	require.NoError(err)
	f, err := os.Create(filepath.Join(tmp, "test.zip"))
	require.NoError(err)
	fmt.Println(f.Name())
	defer f.Close() // nolint: errcheck
	archive := New(f)

	require.Error(archive.Add("nope.txt", "../testdata/nope.txt"))
	require.NoError(archive.Add("foo.txt", "../testdata/foo.txt"))
	require.NoError(archive.Add("sub1", "../testdata/sub1"))
	require.NoError(archive.Add("sub1/bar.txt", "../testdata/sub1/bar.txt"))
	require.NoError(archive.Add("sub1/executable", "../testdata/sub1/executable"))
	require.NoError(archive.Add("sub1/sub2", "../testdata/sub1/sub2"))
	require.NoError(archive.Add("sub1/sub2/subfoo.txt", "../testdata/sub1/sub2/subfoo.txt"))

	require.NoError(archive.Close())
	require.Error(archive.Add("tar.go", "tar.go"))
	require.NoError(f.Close())

	t.Log(f.Name())
	f, err = os.Open(f.Name())
	require.NoError(err)
	defer f.Close() // nolint: errcheck

	info, err := f.Stat()
	require.NoError(err)
	require.Truef(info.Size() < 900, "archived file should be smaller than %d", info.Size())

	r, err := zip.NewReader(f, info.Size())
	require.NoError(err)

	var paths = make([]string, len(r.File))
	for i, zf := range r.File {
		paths[i] = zf.Name
		t.Logf("%s: %v", zf.Name, zf.Mode())
		if zf.Name == "sub1/executable" {
			var ex = zf.Mode() | 0111
			require.Equal(zf.Mode().String(), ex.String())
		}
	}
	require.Equal([]string{
		"foo.txt",
		"sub1/bar.txt",
		"sub1/executable",
		"sub1/sub2/subfoo.txt",
	}, paths)
}
