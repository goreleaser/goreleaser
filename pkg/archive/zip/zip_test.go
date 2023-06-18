package zip

import (
	"archive/zip"
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestZipFile(t *testing.T) {
	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "test.zip"))
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
	require.Truef(t, info.Size() < 1000, "archived file should be smaller than %d", info.Size())

	r, err := zip.NewReader(f, info.Size())
	require.NoError(t, err)

	paths := make([]string, len(r.File))
	for i, zf := range r.File {
		paths[i] = zf.Name
		if zf.Name == "sub1/executable" {
			ex := zf.Mode()&0o111 != 0
			require.True(t, ex, "expected executable permissions, got %s", zf.Mode())
		}
		if zf.Name == "link.txt" {
			require.True(t, zf.FileInfo().Mode()&os.ModeSymlink != 0)
			rc, err := zf.Open()
			require.NoError(t, err)
			var link bytes.Buffer
			_, err = io.Copy(&link, rc)
			require.NoError(t, err)
			rc.Close()
			require.Equal(t, link.String(), "regular.txt")
		}
	}
	require.Equal(t, []string{
		"foo.txt",
		"sub1/bar.txt",
		"sub1/executable",
		"sub1/sub2/subfoo.txt",
		"regular.txt",
		"link.txt",
	}, paths)
}

func TestZipFileInfo(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	f, err := os.Create(filepath.Join(t.TempDir(), "test.zip"))
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

	info, err := f.Stat()
	require.NoError(t, err)

	r, err := zip.NewReader(f, info.Size())
	require.NoError(t, err)

	require.Len(t, r.File, 1)
	for _, next := range r.File {
		require.Equal(t, "nope.txt", next.Name)
		require.Equal(t, now.Unix(), next.Modified.Unix())
		require.Equal(t, fs.FileMode(0o755), next.FileInfo().Mode())
	}
}

func TestTarInvalidLink(t *testing.T) {
	archive := New(io.Discard)
	defer archive.Close() // nolint: errcheck

	require.NoError(t, archive.Add(config.File{
		Source:      "../testdata/badlink.txt",
		Destination: "badlink.txt",
	}))
}

// TODO: add copying test
