package targz

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTarGzFile(t *testing.T) {
	var assert = require.New(t)
	tmp, err := ioutil.TempDir("", "")
	require.NoError(err)
	f, err := os.Create(filepath.Join(tmp, "test.tar.gz"))
	require.NoError(err)
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
	require.Truef(info.Size() < 500, "archived file should be smaller than %d", info.Size())

	gzf, err := gzip.NewReader(f)
	require.NoError(err)
	defer gzf.Close() // nolint: errcheck

	var paths []string
	r := tar.NewReader(gzf)
	for {
		next, err := r.Next()
		if err == io.EOF {
			break
		}
		require.NoError(err)
		paths = append(paths, next.Name)
		t.Logf("%s: %v", next.Name, next.FileInfo().Mode())
		if next.Name == "sub1/executable" {
			var ex = next.FileInfo().Mode() | 0111
			require.Equal(next.FileInfo().Mode().String(), ex.String())
		}
	}
	require.Equal([]string{
		"foo.txt",
		"sub1",
		"sub1/bar.txt",
		"sub1/executable",
		"sub1/sub2",
		"sub1/sub2/subfoo.txt",
	}, paths)
}
