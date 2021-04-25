package targz

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTarGzFile(t *testing.T) {
	var tmp = t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "test.tar.gz"))
	require.NoError(t, err)
	defer f.Close() // nolint: errcheck
	archive := New(f)
	defer archive.Close() // nolint: errcheck

	require.Error(t, archive.Add("nope.txt", "../testdata/nope.txt"))
	require.NoError(t, archive.Add("foo.txt", "../testdata/foo.txt"))
	require.NoError(t, archive.Add("sub1", "../testdata/sub1"))
	require.NoError(t, archive.Add("sub1/bar.txt", "../testdata/sub1/bar.txt"))
	require.NoError(t, archive.Add("sub1/executable", "../testdata/sub1/executable"))
	require.NoError(t, archive.Add("sub1/sub2", "../testdata/sub1/sub2"))
	require.NoError(t, archive.Add("sub1/sub2/subfoo.txt", "../testdata/sub1/sub2/subfoo.txt"))

	require.NoError(t, archive.Close())
	require.Error(t, archive.Add("tar.go", "tar.go"))
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

	var paths []string
	r := tar.NewReader(gzf)
	for {
		next, err := r.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		paths = append(paths, next.Name)
		t.Logf("%s: %v", next.Name, next.FileInfo().Mode())
		if next.Name == "sub1/executable" {
			var ex = next.FileInfo().Mode() | 0111
			require.Equal(t, next.FileInfo().Mode().String(), ex.String())
		}
	}
	require.Equal(t, []string{
		"foo.txt",
		"sub1",
		"sub1/bar.txt",
		"sub1/executable",
		"sub1/sub2",
		"sub1/sub2/subfoo.txt",
	}, paths)
}
