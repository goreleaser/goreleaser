package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootCmdHelp(t *testing.T) {
	var mem = &exitMemento{}
	var cmd = NewRootCmd("foo", mem.Exit).cmd
	cmd.SetArgs([]string{"-h"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, 0, mem.code)
}

func TestRootCmdVersion(t *testing.T) {
	var b bytes.Buffer
	var mem = &exitMemento{}
	var cmd = NewRootCmd("foo", mem.Exit).cmd
	cmd.SetOut(&b)
	cmd.SetArgs([]string{"-v"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "goreleaser version foo\n", b.String())
	require.Equal(t, 0, mem.code)
}

func TestRootRelease(t *testing.T) {
	_, back := setup(t)
	defer back()
	var mem = &exitMemento{}
	var b bytes.Buffer
	var cmd = NewRootCmd("foo", mem.Exit)
	cmd.cmd.SetOut(&b)
	cmd.cmd.SetErr(&b)
	cmd.Execute([]string{})
	require.Contains(t, "releasing...", b.String())
	require.Contains(t, "release failed after", b.String())
	require.Contains(t, "error=github/gitlab/gitea releases: failed to publish artifacts", b.String())
	require.Equal(t, 1, mem.code)
}

func TestRootReleaseDebug(t *testing.T) {
	_, back := setup(t)
	defer back()
	var mem = &exitMemento{}
	var b bytes.Buffer
	var cmd = NewRootCmd("foo", mem.Exit)
	cmd.cmd.SetOut(&b)
	cmd.cmd.SetErr(&b)
	cmd.Execute([]string{"r", "--debug"})
	require.Contains(t, "debug logs enabled", b.String())
	require.Contains(t, "releasing...", b.String())
	require.Contains(t, "release failed after", b.String())
	require.Contains(t, "error=github/gitlab/gitea releases: failed to publish artifacts", b.String())
	require.Equal(t, 1, mem.code)
}

type exitMemento struct {
	code int
}

func (e *exitMemento) Exit(i int) {
	e.code = i
}
