//go:build integration

package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegrationBuild(t *testing.T) {
	setup(t)
	cmd := newBuildCmd()
	cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2", "--deprecated"})
	require.NoError(t, cmd.cmd.Execute())
}

func TestIntegrationBuildAutoSnapshot(t *testing.T) {
	t.Run("clean", func(t *testing.T) {
		setup(t)
		cmd := newBuildCmd()
		cmd.cmd.SetArgs([]string{"--auto-snapshot"})
		require.NoError(t, cmd.cmd.Execute())
		matches, err := filepath.Glob("./dist/fake_*/fake")
		require.NoError(t, err)
		require.Len(t, matches, 1)
	})

	t.Run("dirty", func(t *testing.T) {
		setup(t)
		createFile(t, "foo", "force dirty tree")
		cmd := newBuildCmd()
		cmd.cmd.SetArgs([]string{"--auto-snapshot"})
		require.NoError(t, cmd.cmd.Execute())
		matches, err := filepath.Glob("./dist/fake_*/fake_snapshot")
		require.NoError(t, err)
		require.Len(t, matches, 1)
	})
}

func TestIntegrationBuildSingleTarget(t *testing.T) {
	setup(t)
	cmd := newBuildCmd()
	cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2", "--deprecated", "--single-target"})
	require.NoError(t, cmd.cmd.Execute())
}

func TestIntegrationBuildInvalidConfig(t *testing.T) {
	setup(t)
	createFile(t, "goreleaser.yml", "version: 2\nfoo: bar")
	cmd := newBuildCmd()
	cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2", "--deprecated"})
	require.EqualError(t, cmd.cmd.Execute(), "yaml: unmarshal errors:\n  line 2: field foo not found in type config.Project")
}

func TestIntegrationBuildBrokenProject(t *testing.T) {
	setup(t)
	createFile(t, "main.go", "not a valid go file")
	cmd := newBuildCmd()
	cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2"})
	require.ErrorContains(t, cmd.cmd.Execute(), "failed to parse dir: .: main.go:1:1: expected 'package', found not")
}
