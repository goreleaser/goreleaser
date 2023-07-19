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

func TestInitConfigAlreadyExist(t *testing.T) {
	folder := setupInitTest(t)
	config := "foo.yaml"
	configPath := filepath.Join(folder, config)

	cmd := newInitCmd().cmd
	cmd.SetArgs([]string{"-f", config})
	content := []byte("foo: bar\n")
	require.NoError(t, os.WriteFile(configPath, content, 0o644))

	require.Error(t, cmd.Execute())
	require.FileExists(t, configPath)
	require.NoFileExists(t, filepath.Join(folder, ".gitignore"))

	bts, err := os.ReadFile(configPath)
	require.NoError(t, err)
	require.Equal(t, string(content), string(bts))
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

func TestHasDistIgnored(t *testing.T) {
	t.Run("ignored", func(t *testing.T) {
		require.True(t, hasDistIgnored("../.gitignore"))
	})

	t.Run("ignored middle of file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "gitignore")
		require.NoError(t, os.WriteFile(path, []byte("foo\ndist/\nbar\n"), 0o644))
		require.True(t, hasDistIgnored(path))
	})

	t.Run("not ignored", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "gitignore")
		require.NoError(t, os.WriteFile(path, []byte("foo\nbar\n"), 0o644))
		require.False(t, hasDistIgnored(path))
	})

	t.Run("file does not exist", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "gitignore")
		require.False(t, hasDistIgnored(path))
	})
}
