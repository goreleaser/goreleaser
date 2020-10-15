package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompletionGeneration(t *testing.T) {
	for _, shell := range []string{"bash", "zsh"} {
		completionCmd := newCompletionCmd().cmd
		stdout := bytes.NewBufferString("")
		stderr := bytes.NewBufferString("")
		completionCmd.SetOut(stdout)
		completionCmd.SetErr(stderr)
		completionCmd.SetArgs([]string{shell})
		err := completionCmd.Execute()
		require.NoError(t, err, shell+" arg experienced error with goreleaser completion:\n"+stderr.String())
		require.Equal(t, "", stderr.String(), shell+" arg experienced error with goreleaser completion:\n"+stderr.String())
		require.NotEmpty(t, stdout.String(), shell+" arg reported nothing to stdout with goreleaser completion")
	}
}
