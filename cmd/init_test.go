package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	folder := setupInitTest(t)
	cmd := newInitCmd().cmd
	config := "foo.yaml"
	cmd.SetArgs([]string{"-f", config})
	require.NoError(t, cmd.Execute())
	require.FileExists(t, filepath.Join(folder, config))
	require.FileExists(t, filepath.Join(folder, ".gitignore"))
}

func TestInitGitIgnoreExists(t *testing.T) {
	folder := setupInitTest(t)
	cmd := newInitCmd().cmd

	config := "foo.yaml"
	require.NoError(t, os.WriteFile(filepath.Join(folder, ".gitignore"), []byte("mybinary\n"), 0o644))

	cmd.SetArgs([]string{"-f", config})
	require.NoError(t, cmd.Execute())
	require.FileExists(t, filepath.Join(folder, config))
	require.FileExists(t, filepath.Join(folder, ".gitignore"))

	bts, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	require.Equal(t, "mybinary\n\ndist/\n", string(bts))
}

func TestInitFileExists(t *testing.T) {
	folder := setupInitTest(t)
	cmd := newInitCmd().cmd
	path := filepath.Join(folder, "twice.yaml")
	cmd.SetArgs([]string{"-f", path})
	require.NoError(t, cmd.Execute())
	require.EqualError(t, cmd.Execute(), "open "+path+": file exists")
	require.FileExists(t, path)
}

func TestInitFileError(t *testing.T) {
	folder := setupInitTest(t)
	cmd := newInitCmd().cmd
	path := filepath.Join(folder, "nope.yaml")
	require.NoError(t, os.Chmod(folder, 0o000))
	cmd.SetArgs([]string{"-f", path})
	require.EqualError(t, cmd.Execute(), "open "+path+": permission denied")
}

func setupInitTest(tb testing.TB) string {
	tb.Helper()

	folder := tb.TempDir()
	wd, err := os.Getwd()
	require.NoError(tb, err)
	tb.Cleanup(func() {
		require.NoError(tb, os.Chdir(wd))
	})
	require.NoError(tb, os.Chdir(folder))
	return folder
}
