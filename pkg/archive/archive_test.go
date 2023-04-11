package archive

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestArchive(t *testing.T) {
	folder := t.TempDir()
	empty, err := os.Create(folder + "/empty.txt")
	require.NoError(t, err)
	require.NoError(t, empty.Close())
	require.NoError(t, os.Mkdir(folder+"/folder-inside", 0o755))

	for _, format := range []string{"tar.gz", "zip", "gz", "tar.xz", "tar", "tgz", "txz"} {
		format := format
		t.Run(format, func(t *testing.T) {
			f1, err := os.Create(filepath.Join(t.TempDir(), "1.tar"))
			require.NoError(t, err)

			archive, err := New(f1, format)
			require.NoError(t, err)
			require.NoError(t, archive.Add(config.File{
				Source:      empty.Name(),
				Destination: "empty.txt",
			}))
			require.Error(t, archive.Add(config.File{
				Source:      empty.Name() + "_nope",
				Destination: "dont.txt",
			}))
			require.NoError(t, archive.Close())
			require.NoError(t, f1.Close())

			if format == "tar.xz" || format == "txz" || format == "gz" {
				_, err := Copying(f1, io.Discard, format)
				require.Error(t, err)
				return
			}

			f1, err = os.Open(f1.Name())
			require.NoError(t, err)
			f2, err := os.Create(filepath.Join(t.TempDir(), "2.tar"))
			require.NoError(t, err)

			a, err := Copying(f1, f2, format)
			require.NoError(t, err)
			require.NoError(t, f1.Close())

			require.NoError(t, a.Add(config.File{
				Source:      empty.Name(),
				Destination: "added_later.txt",
			}))
			require.NoError(t, a.Add(config.File{
				Source:      empty.Name(),
				Destination: "ملف.txt",
			}))
			require.NoError(t, a.Close())
			require.NoError(t, f2.Close())

			require.ElementsMatch(
				t,
				[]string{"empty.txt", "added_later.txt", "ملف.txt"},
				testlib.LsArchive(t, f2.Name(), format),
			)
		})
	}

	// unsupported format...
	t.Run("7z", func(t *testing.T) {
		_, err := New(io.Discard, "7z")
		require.EqualError(t, err, "invalid archive format: 7z")
	})
}
