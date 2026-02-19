//go:build integration

package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegrationRelease(t *testing.T) {
	setup(t)
	cmd := newReleaseCmd()
	cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2", "--deprecated"})
	require.NoError(t, cmd.cmd.Execute())
}

func TestIntegrationReleaseAutoSnapshot(t *testing.T) {
	t.Run("clean", func(t *testing.T) {
		setup(t)
		cmd := newReleaseCmd()
		cmd.cmd.SetArgs([]string{"--auto-snapshot", "--skip=publish"})
		require.NoError(t, cmd.cmd.Execute())
		require.FileExists(t, "dist/fake_0.0.2_checksums.txt", "should have created checksums when run with --snapshot")
	})

	t.Run("dirty", func(t *testing.T) {
		setup(t)
		createFile(t, "foo", "force dirty tree")
		cmd := newReleaseCmd()
		cmd.cmd.SetArgs([]string{"--auto-snapshot", "--skip=publish"})
		require.NoError(t, cmd.cmd.Execute())
		matches, err := filepath.Glob("./dist/fake_0.0.2-SNAPSHOT-*_checksums.txt")
		require.NoError(t, err)
		require.Len(t, matches, 1, "should have implied --snapshot")
	})
}

func TestIntegrationReleaseInvalidConfig(t *testing.T) {
	setup(t)
	createFile(t, "goreleaser.yml", "foo: bar\nversion: 2")
	cmd := newReleaseCmd()
	cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2", "--deprecated"})
	require.EqualError(t, cmd.cmd.Execute(), "yaml: unmarshal errors:\n  line 1: field foo not found in type config.Project")
}

func TestIntegrationReleaseBrokenProject(t *testing.T) {
	setup(t)
	createFile(t, "main.go", "not a valid go file")
	cmd := newReleaseCmd()
	cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2"})
	require.ErrorContains(t, cmd.cmd.Execute(), "failed to parse dir: .: main.go:1:1: expected 'package', found not")
}
