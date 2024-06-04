package cmd

import (
	"bytes"
	"strings"
	"testing"

	goversion "github.com/caarlos0/go-version"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

var testversion = goversion.Info{
	GitVersion: "1.2.3",
}

func TestRootCmd(t *testing.T) {
	mem := &exitMemento{}
	Execute(testversion, mem.Exit, []string{"-h"})
	require.Equal(t, 0, mem.code)
}

func TestRootCmdHelp(t *testing.T) {
	mem := &exitMemento{}
	cmd := newRootCmd(testversion, mem.Exit).cmd
	cmd.SetArgs([]string{"-h"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, 0, mem.code)
}

func TestRootCmdVersion(t *testing.T) {
	var b bytes.Buffer
	mem := &exitMemento{}
	cmd := newRootCmd(testversion, mem.Exit).cmd
	cmd.SetOut(&b)
	cmd.SetArgs([]string{"-v"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, b.String(), "1.2.3")
	require.Equal(t, 0, mem.code)
}

func TestRootCmdExitCode(t *testing.T) {
	mem := &exitMemento{}
	cmd := newRootCmd(testversion, mem.Exit)
	args := []string{"check", "--deprecated", "-f", "testdata/good.yml"}
	cmd.Execute(args)
	require.Equal(t, 2, mem.code)
}

func TestRootRelease(t *testing.T) {
	setup(t)
	mem := &exitMemento{}
	cmd := newRootCmd(testversion, mem.Exit)
	cmd.Execute([]string{})
	require.Equal(t, 1, mem.code)
}

func TestRootReleaseVerbose(t *testing.T) {
	setup(t)
	mem := &exitMemento{}
	cmd := newRootCmd(testversion, mem.Exit)
	cmd.Execute([]string{"r", "--verbose"})
	require.Equal(t, 1, mem.code)
}

func TestShouldPrependRelease(t *testing.T) {
	result := func(args []string) bool {
		return shouldPrependRelease(newRootCmd(testversion, func(_ int) {}).cmd, args)
	}

	t.Run("no args", func(t *testing.T) {
		require.True(t, result([]string{}))
	})

	t.Run("release args", func(t *testing.T) {
		require.True(t, result([]string{"--skip=validate"}))
	})

	t.Run("several release args", func(t *testing.T) {
		require.True(t, result([]string{"--skip=validate", "--snapshot"}))
	})

	for _, s := range []string{"--help", "-h", "-v", "--version"} {
		t.Run(s, func(t *testing.T) {
			require.False(t, result([]string{s}))
		})
	}

	t.Run("check", func(t *testing.T) {
		require.False(t, result([]string{"check", "-f", "testdata/good.yml"}))
	})

	t.Run("help", func(t *testing.T) {
		require.False(t, result([]string{"help"}))
	})

	t.Run("__complete", func(t *testing.T) {
		require.False(t, result([]string{"__complete"}))
	})

	t.Run("__completeNoDesc", func(t *testing.T) {
		require.False(t, result([]string{"__completeNoDesc"}))
	})
}

func TestShouldDisableLogs(t *testing.T) {
	testCases := []struct {
		args   []string
		expect bool
	}{
		{nil, false},
		{[]string{"release"}, false},
		{[]string{"release", "--clean"}, false},
		{[]string{"help"}, true},
		{[]string{"completion"}, true},
		{[]string{"man"}, true},
		{[]string{"jsonschema"}, true},
		{[]string{"docs"}, true},
		{[]string{cobra.ShellCompRequestCmd}, true},
		{[]string{cobra.ShellCompNoDescRequestCmd}, true},
	}
	for _, tC := range testCases {
		t.Run(strings.Join(tC.args, " "), func(t *testing.T) {
			require.Equal(t, tC.expect, shouldDisableLogs(tC.args))
		})
	}
}
