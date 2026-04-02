package xz_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	xza "github.com/goreleaser/goreleaser/v2/pkg/archive/xz"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
	"github.com/ulikunitz/xz"
)

func TestXzFile(t *testing.T) {
	tmp := t.TempDir()

	f, err := os.Create(filepath.Join(tmp, "test.xz"))
	require.NoError(t, err)
	defer f.Close()

	archive := xza.New(f)
	defer archive.Close()

	require.NoError(t, archive.Add(config.File{
		Destination: "sub1/sub2/subfoo.txt",
		Source:      "../testdata/sub1/sub2/subfoo.txt",
	}))
	require.EqualError(t, archive.Add(config.File{
		Destination: "foo.txt",
		Source:      "../testdata/foo.txt",
	}), "xz: failed to add foo.txt, only one file can be archived in xz format")
	require.NoError(t, archive.Close())
	require.NoError(t, f.Close())

	f, err = os.Open(f.Name())
	require.NoError(t, err)
	defer f.Close()

	info, err := f.Stat()
	require.NoError(t, err)
	require.Lessf(t, info.Size(), int64(500), "archived file should be smaller than %d", info.Size())

	xzf, err := xz.NewReader(f)
	require.NoError(t, err)

	bts, err := io.ReadAll(xzf)
	require.NoError(t, err)
	require.Equal(t, "sub\n", string(bts))
}
