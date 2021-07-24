package gio

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCopy(t *testing.T) {
	tmp := t.TempDir()
	a := "testdata/somefile.txt"
	b := tmp + "/somefile.txt"
	require.NoError(t, CopyFileWithSrcMode(a, b))
	equal, err := EqualFiles(a, b)
	require.NoError(t, err)
	require.True(t, equal)
}


func TestEqualFilesModeChanged(t *testing.T) {
	tmp := t.TempDir()
	a := "testdata/somefile.txt"
	b := tmp + "/somefile.txt"
	require.NoError(t, CopyFileWithSrcMode(a, b))
	require.NoError(t,os.Chmod(b, 0755))
	equal, err := EqualFiles(a, b)
	require.NoError(t, err)
	require.False(t, equal)
}

func TestEqualFilesContentsChanged(t *testing.T) {
	tmp := t.TempDir()
	a := "testdata/somefile.txt"
	b := tmp + "/somefile.txt"
	require.NoError(t, CopyFileWithSrcMode(a, b))
	require.NoError(t,os.WriteFile(b, []byte("hello world"), 0644))
	equal, err := EqualFiles(a, b)
	require.NoError(t, err)
	require.False(t, equal)
}

func TestEqualFilesDontExist(t *testing.T) {
	a := "testdata/nope.txt"
	b := "testdata/somefile.txt"
	c:="testdata/notadir/lala"
	require.Error(t, CopyFileWithSrcMode(a,b))
	require.Error(t, CopyFile(a,b, 0644))
	require.Error(t, CopyFileWithSrcMode(b,c))
}
