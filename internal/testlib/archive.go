package testlib

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ulikunitz/xz"
)

// LsArchive return the file list of a given archive in a given formatkj
func LsArchive(tb testing.TB, path, format string) []string {
	tb.Helper()
	f := openFile(tb, path)
	switch format {
	case "tar.gz", "tgz":
		return doLsTar(openGzip(tb, f))
	case "tar.xz", "txz":
		return doLsTar(openXz(tb, f))
	case "tar":
		return doLsTar(f)
	case "zip":
		return lsZip(tb, f)
	case "gz":
		return []string{openGzip(tb, f).Header.Name}
	default:
		tb.Errorf("invalid format: %s", format)
		return nil
	}
}

func openGzip(tb testing.TB, r io.Reader) *gzip.Reader {
	tb.Helper()
	gz, err := gzip.NewReader(r)
	require.NoError(tb, err)
	return gz
}

func openXz(tb testing.TB, r io.Reader) *xz.Reader {
	tb.Helper()
	xz, err := xz.NewReader(r)
	require.NoError(tb, err)
	return xz
}

func lsZip(tb testing.TB, f *os.File) []string {
	tb.Helper()

	stat, err := f.Stat()
	require.NoError(tb, err)
	z, err := zip.NewReader(f, stat.Size())
	require.NoError(tb, err)

	var paths []string
	for _, zf := range z.File {
		paths = append(paths, zf.Name)
	}
	return paths
}

func doLsTar(f io.Reader) []string {
	z := tar.NewReader(f)
	var paths []string
	for {
		h, err := z.Next()
		if h == nil || err == io.EOF {
			break
		}
		if h.Format == tar.FormatPAX {
			continue
		}
		paths = append(paths, h.Name)
	}
	return paths
}

func openFile(tb testing.TB, path string) *os.File {
	tb.Helper()
	f, err := os.Open(path)
	require.NoError(tb, err)
	return f
}
