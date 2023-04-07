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

func LsArchive(tb testing.TB, path, format string) []string {
	tb.Helper()
	switch format {
	case "tar.gz", "tgz":
		return LsTarGz(tb, path)
	case "tar.xz", "txz":
		return LsTarXz(tb, path)
	case "tar":
		return LsTar(tb, path)
	case "zip":
		return LsZip(tb, path)
	case "gz":
		return LsGz(tb, path)
	default:
		tb.Errorf("invalid format: %s", format)
		return nil
	}
}

func LsTar(tb testing.TB, path string) []string {
	tb.Helper()
	return doLsTar(openFile(tb, path))
}

func LsTarGz(tb testing.TB, path string) []string {
	tb.Helper()

	gz, err := gzip.NewReader(openFile(tb, path))
	require.NoError(tb, err)
	return doLsTar(gz)
}

func LsTarXz(tb testing.TB, path string) []string {
	tb.Helper()

	gz, err := xz.NewReader(openFile(tb, path))
	require.NoError(tb, err)
	return doLsTar(gz)
}

func LsZip(tb testing.TB, path string) []string {
	tb.Helper()

	stat, err := os.Stat(path)
	require.NoError(tb, err)
	z, err := zip.NewReader(openFile(tb, path), stat.Size())
	require.NoError(tb, err)

	var paths []string
	for _, zf := range z.File {
		paths = append(paths, zf.Name)
	}
	return paths
}

func LsGz(tb testing.TB, path string) []string {
	z, err := gzip.NewReader(openFile(tb, path))
	require.NoError(tb, err)

	return []string{z.Header.Name}
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
