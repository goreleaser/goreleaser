package gio

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCopy(t *testing.T) {
	tmp := t.TempDir()
	a := "testdata/somefile.txt"
	b := tmp + "/somefile.txt"
	require.NoError(t, Copy(a, b))
	requireEqualFiles(t, a, b)
}

func TestEqualFilesModeChanged(t *testing.T) {
	tmp := t.TempDir()
	a := "testdata/somefile.txt"
	b := tmp + "/somefile.txt"
	require.NoError(t, CopyWithMode(a, b, 0o755))
	requireNotEqualFiles(t, a, b)
}

func TestEqualFilesContentsChanged(t *testing.T) {
	tmp := t.TempDir()
	a := "testdata/somefile.txt"
	b := tmp + "/somefile.txt"
	require.NoError(t, Copy(a, b))
	require.NoError(t, os.WriteFile(b, []byte("hello world"), 0o644))
	requireNotEqualFiles(t, a, b)
}

func TestEqualFilesDontExist(t *testing.T) {
	a := "testdata/nope.txt"
	b := "testdata/somefile.txt"
	c := "testdata/notadir/lala"
	require.Error(t, Copy(a, b))
	require.Error(t, CopyWithMode(a, b, 0o644))
	require.Error(t, Copy(b, c))
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src, err := ioutil.TempFile(dir, "src")
	require.NoError(t, err)
	require.NoError(t, src.Close())
	dst := filepath.Join(dir, "dst")
	require.NoError(t, os.WriteFile(src.Name(), []byte("foo"), 0o644))
	require.NoError(t, Copy(src.Name(), dst))
	requireEqualFiles(t, src.Name(), dst)
}

func TestCopyDirectory(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	const testFile = "test"
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, testFile), []byte("foo"), 0o644))
	require.NoError(t, Copy(srcDir, dstDir))
	requireEqualFiles(t, filepath.Join(srcDir, testFile), filepath.Join(dstDir, testFile))
}

func TestCopyTwoLevelDirectory(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	srcLevel2 := filepath.Join(srcDir, "level2")
	const testFile = "test"

	require.NoError(t, os.Mkdir(srcLevel2, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, testFile), []byte("foo"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcLevel2, testFile), []byte("foo"), 0o644))

	require.NoError(t, Copy(srcDir, dstDir))

	requireEqualFiles(t, filepath.Join(srcDir, testFile), filepath.Join(dstDir, testFile))
	requireEqualFiles(t, filepath.Join(srcLevel2, testFile), filepath.Join(dstDir, "level2", testFile))
}

func requireEqualFiles(tb testing.TB, a, b string) {
	tb.Helper()
	eq, err := EqualFiles(a, b)
	require.NoError(tb, err)
	require.True(tb, eq, "%s != %s", a, b)
}

func requireNotEqualFiles(tb testing.TB, a, b string) {
	tb.Helper()
	eq, err := EqualFiles(a, b)
	require.NoError(tb, err)
	require.False(tb, eq, "%s == %s", a, b)
}
