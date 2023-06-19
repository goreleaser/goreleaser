package tar

import (
	"archive/tar"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestTarFile(t *testing.T) {
	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "test.tar"))
	require.NoError(t, err)
	defer f.Close() // nolint: errcheck
	archive := New(f)
	defer archive.Close() // nolint: errcheck

	require.Error(t, archive.Add(config.File{
		Source:      "../testdata/nope.txt",
		Destination: "nope.txt",
	}))
	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/foo.txt",
		Destination: "foo.txt",
	}))
	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/sub1",
		Destination: "sub1",
	}))
	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/sub1/bar.txt",
		Destination: "sub1/bar.txt",
	}))
	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/sub1/executable",
		Destination: "sub1/executable",
	}))
	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/sub1/sub2",
		Destination: "sub1/sub2",
	}))
	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/sub1/sub2/subfoo.txt",
		Destination: "sub1/sub2/subfoo.txt",
	}))
	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/regular.txt",
		Destination: "regular.txt",
	}))
	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/link.txt",
		Destination: "link.txt",
	}))

	require.ErrorIs(t, archive.Add(config.File{
		Source:      "../testdata/regular.txt",
		Destination: "link.txt",
	}), fs.ErrExist)

	require.NoError(t, archive.Close())
	require.Error(t, archive.Add(config.File{
		Source:      "tar.go",
		Destination: "tar.go",
	}))
	require.NoError(t, f.Close())

	f, err = os.Open(f.Name())
	require.NoError(t, err)
	defer f.Close() // nolint: errcheck

	info, err := f.Stat()
	require.NoError(t, err)
	require.Truef(t, info.Size() < 10000, "archived file should be smaller than %d", info.Size())

	var paths []string
	r := tar.NewReader(f)
	for {
		next, err := r.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		paths = append(paths, next.Name)
		if next.Name == "sub1/executable" {
			ex := next.FileInfo().Mode()&0o111 != 0
			require.True(t, ex, "expected executable permissions, got %s", next.FileInfo().Mode())
		}
		if next.Name == "link.txt" {
			require.Equal(t, next.Linkname, "regular.txt")
		}
	}
	require.Equal(t, []string{
		"foo.txt",
		"sub1",
		"sub1/bar.txt",
		"sub1/executable",
		"sub1/sub2",
		"sub1/sub2/subfoo.txt",
		"regular.txt",
		"link.txt",
	}, paths)
}

func TestTarFileInfo(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	f, err := os.Create(filepath.Join(t.TempDir(), "test.tar"))
	require.NoError(t, err)
	defer f.Close() // nolint: errcheck
	archive := New(f)
	defer archive.Close() // nolint: errcheck

	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/foo.txt",
		Destination: "nope.txt",
		Info: config.FileInfo{
			Mode:        0o755,
			Owner:       "carlos",
			Group:       "root",
			ParsedMTime: now,
		},
	}))

	require.NoError(t, archive.Close())
	require.NoError(t, f.Close())

	f, err = os.Open(f.Name())
	require.NoError(t, err)
	defer f.Close() // nolint: errcheck

	var found int
	r := tar.NewReader(f)
	for {
		next, err := r.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		found++
		require.Equal(t, "nope.txt", next.Name)
		require.Equal(t, now, next.ModTime)
		require.Equal(t, fs.FileMode(0o755), next.FileInfo().Mode())
		require.Equal(t, "carlos", next.Uname)
		require.Equal(t, 0, next.Uid)
		require.Equal(t, "root", next.Gname)
		require.Equal(t, 0, next.Gid)
	}
	require.Equal(t, 1, found)
}

func TestTarInvalidLink(t *testing.T) {
	archive := New(io.Discard)
	defer archive.Close() // nolint: errcheck

	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/badlink.txt",
		Destination: "badlink.txt",
	}))
}

func TestCopying(t *testing.T) {
	f1, err := os.Create(filepath.Join(t.TempDir(), "1.tar"))
	require.NoError(t, err)
	f2, err := os.Create(filepath.Join(t.TempDir(), "2.tar"))
	require.NoError(t, err)

	t1 := New(f1)
	require.NoError(t, t1.Add(config.File{
		Source:      "../testdata/foo.txt",
		Destination: "foo.txt",
	}))
	require.NoError(t, t1.Add(config.File{
		Source:      "../testdata/foo.txt",
		Destination: "ملف.txt",
	}))
	require.NoError(t, t1.Close())
	require.NoError(t, f1.Close())

	f1, err = os.Open(f1.Name())
	require.NoError(t, err)

	t2, err := Copying(f1, f2)
	require.NoError(t, err)
	require.NoError(t, t2.Add(config.File{
		Source:      "../testdata/sub1/executable",
		Destination: "executable",
	}))
	require.NoError(t, t2.Add(config.File{
		Source:      "../testdata/sub1/executable",
		Destination: "ملف.exe",
	}))
	require.NoError(t, t2.Close())
	require.NoError(t, f2.Close())
	require.NoError(t, f1.Close())

	require.Equal(t, []string{"foo.txt", "ملف.txt"}, testlib.LsArchive(t, f1.Name(), "tar"))
	require.Equal(t, []string{"foo.txt", "ملف.txt", "executable", "ملف.exe"}, testlib.LsArchive(t, f2.Name(), "tar"))
}
