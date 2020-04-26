package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootCmdHelp(t *testing.T) {
	var cmd = NewRootCmd("foo").cmd
	cmd.SetArgs([]string{"-h"})
	require.NoError(t, cmd.Execute())
}

func TestRootCmdVersion(t *testing.T) {
	var b bytes.Buffer
	var cmd = NewRootCmd("foo").cmd
	cmd.SetOut(&b)
	cmd.SetArgs([]string{"-v"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "goreleaser version foo\n", b.String())
}
