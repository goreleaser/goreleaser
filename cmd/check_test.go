package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckConfig(t *testing.T) {
	cmd := newCheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/good.yml"})
	require.NoError(t, cmd.cmd.Execute())
}

func TestCheckConfigThatDoesNotExist(t *testing.T) {
	cmd := newCheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/nope.yml"})
	require.ErrorIs(t, cmd.cmd.Execute(), os.ErrNotExist)
}

func TestCheckConfigUnmarshalError(t *testing.T) {
	cmd := newCheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/unmarshal_error.yml"})
	require.EqualError(t, cmd.cmd.Execute(), "yaml: unmarshal errors:\n  line 1: field foo not found in type config.Project")
}

func TestCheckConfigInvalid(t *testing.T) {
	cmd := newCheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/invalid.yml"})
	require.Error(t, cmd.cmd.Execute())
}

func TestCheckConfigInvalidQuiet(t *testing.T) {
	cmd := newCheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/invalid.yml", "-q"})
	require.Error(t, cmd.cmd.Execute())
}

func TestCheckConfigDeprecated(t *testing.T) {
	cmd := newCheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/good.yml", "--deprecated"})
	require.Error(t, cmd.cmd.Execute())
}
