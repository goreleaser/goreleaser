package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	var folder = mktemp(t)
	var cmd = newInitCmd().cmd
	var path = filepath.Join(folder, "foo.yaml")
	cmd.SetArgs([]string{"-f", path})
	require.NoError(t, cmd.Execute())
	require.FileExists(t, path)
}

func TestInitFileExists(t *testing.T) {
	var folder = mktemp(t)
	var cmd = newInitCmd().cmd
	var path = filepath.Join(folder, "twice.yaml")
	cmd.SetArgs([]string{"-f", path})
	require.NoError(t, cmd.Execute())
	require.EqualError(t, cmd.Execute(), "open "+path+": file exists")
	require.FileExists(t, path)
}

func TestInitFileError(t *testing.T) {
	var folder = mktemp(t)
	var cmd = newInitCmd().cmd
	var path = filepath.Join(folder, "nope.yaml")
	require.NoError(t, os.Chmod(folder, 0000))
	cmd.SetArgs([]string{"-f", path})
	require.EqualError(t, cmd.Execute(), "open "+path+": permission denied")
}

func mktemp(t *testing.T) string {
	folder, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	return folder
}
