package gio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/stretchr/testify/require"
)

func TestCopy(t *testing.T) {
	tmp := t.TempDir()
	a := "testdata/somefile.txt"
	b := tmp + "/somefile.txt"
	require.NoError(t, Copy(a, b))
	requireEqualFiles(t, a, b)
}

func TestCopySymlink(t *testing.T) {
	tmp := t.TempDir()
	a := "testdata/somefile.txt"
	b := tmp + "/somefile.txt"
	c := tmp + "/somefile2.txt"
	require.NoError(t, os.Symlink(a, b))
	require.NoError(t, Copy(b, c))

	fi, err := os.Lstat(c)
	require.NoError(t, err)
	require.NotEqual(t, 0, fi.Mode()&os.ModeSymlink)

	l, err := os.Readlink(c)
	require.NoError(t, err)
	require.Equal(t, a, filepath.ToSlash(l))
}

func TestCopyDirectorySymlinkWithTrailingSeparator(t *testing.T) {
	testlib.SkipIfWindows(t, "trailing slash follows directory symlinks on Unix")
	t.Parallel()
	src := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "asset.txt"), []byte("asset"), 0o644))
	link := filepath.Join(t.TempDir(), "assets")
	require.NoError(t, os.Symlink(src, link))
	dst := t.TempDir()

	require.NoError(t, Copy(link+string(os.PathSeparator), dst))
	data, err := os.ReadFile(filepath.Join(dst, "asset.txt"))
	require.NoError(t, err)
	require.Equal(t, "asset", string(data))
	info, err := os.Lstat(dst)
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestEqualFilesModeChanged(t *testing.T) {
	testlib.SkipIfWindows(t)
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

func TestCopyErrors(t *testing.T) {
	a := "testdata/nope.txt"
	b := "testdata/also-nope.txt"

	err := copySymlink(a, b)
	require.Error(t, err)

	err = copyFile(a, b, 0o755)
	require.Error(t, err)
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src, err := os.CreateTemp(dir, "src")
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

func TestCopyDotRelativeDirectory(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	require.NoError(t, os.MkdirAll(filepath.Join("assets", "css"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join("assets", "index.html"), []byte("index"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join("assets", "css", "style.css"), []byte("style"), 0o644))
	dstDir := t.TempDir()

	require.NoError(t, Copy("./assets", filepath.Join(dstDir, "assets")))

	requireEqualFiles(t, filepath.Join("assets", "index.html"), filepath.Join(dstDir, "assets", "index.html"))
	requireEqualFiles(t, filepath.Join("assets", "css", "style.css"), filepath.Join(dstDir, "assets", "css", "style.css"))
	require.NoFileExists(t, filepath.Join(dstDir, "assets", "assets", "index.html"))
}

func TestCopyDotDirectory(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	require.NoError(t, os.WriteFile("config.json", []byte("config"), 0o644))
	require.NoError(t, os.WriteFile(".dockerignore", []byte("dist"), 0o644))
	dstDir := t.TempDir()

	require.NoError(t, Copy(".", dstDir))

	requireEqualFiles(t, "config.json", filepath.Join(dstDir, "config.json"))
	requireEqualFiles(t, ".dockerignore", filepath.Join(dstDir, ".dockerignore"))
	require.NoFileExists(t, filepath.Join(dstDir, "configjson"))
	require.NoFileExists(t, filepath.Join(dstDir, "dockerignore"))
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
