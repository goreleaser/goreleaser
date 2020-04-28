package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootCmd(t *testing.T) {
	var mem = &exitMemento{}
	Execute("1.2.3", mem.Exit, []string{"-h"})
	require.Equal(t, 0, mem.code)
}

func TestRootCmdHelp(t *testing.T) {
	var mem = &exitMemento{}
	var cmd = newRootCmd("", mem.Exit).cmd
	cmd.SetArgs([]string{"-h"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, 0, mem.code)
}

func TestRootCmdVersion(t *testing.T) {
	var b bytes.Buffer
	var mem = &exitMemento{}
	var cmd = newRootCmd("1.2.3", mem.Exit).cmd
	cmd.SetOut(&b)
	cmd.SetArgs([]string{"-v"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "goreleaser version 1.2.3\n", b.String())
	require.Equal(t, 0, mem.code)
}

func TestRootCmdExitCode(t *testing.T) {
	var mem = &exitMemento{}
	var cmd = newRootCmd("", mem.Exit)
	var args = []string{"check", "--deprecated", "-f", "testdata/good.yml"}
	cmd.Execute(args)
	require.Equal(t, 2, mem.code)
}

func TestRootRelease(t *testing.T) {
	_, back := setup(t)
	defer back()
	var mem = &exitMemento{}
	var cmd = newRootCmd("", mem.Exit)
	cmd.Execute([]string{})
	require.Equal(t, 1, mem.code)
}

func TestRootReleaseDebug(t *testing.T) {
	_, back := setup(t)
	defer back()
	var mem = &exitMemento{}
	var cmd = newRootCmd("", mem.Exit)
	cmd.Execute([]string{"r", "--debug"})
	require.Equal(t, 1, mem.code)
}

func TestShouldPrependRelease(t *testing.T) {
	var result = func(args []string) bool {
		return shouldPrependRelease(newRootCmd("1", func(_ int) {}).cmd, args)
	}

	t.Run("no args", func(t *testing.T) {
		require.True(t, result([]string{}))
	})

	t.Run("release args", func(t *testing.T) {
		require.True(t, result([]string{"--skip-validate"}))
	})

	t.Run("several release args", func(t *testing.T) {
		require.True(t, result([]string{"--skip-validate", "--snapshot"}))
	})

	for _, s := range []string{"--help", "-h", "-v", "--version"} {
		t.Run(s, func(t *testing.T) {
			require.False(t, result([]string{s}))
		})
	}

	t.Run("check", func(t *testing.T) {
		require.False(t, result([]string{"check", "-f", "testdata/good.yml"}))
	})
}
