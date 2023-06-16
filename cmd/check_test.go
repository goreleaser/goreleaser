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

func TestCheckConfigNoArgs(t *testing.T) {
	cmd := newCheckCmd()
	cmd.cmd.SetArgs(nil)
	require.NoError(t, cmd.cmd.Execute())
	require.Equal(t, 1, cmd.checked)
}

func TestCheckConfigMultipleFiles(t *testing.T) {
	cmd := newCheckCmd()
	cmd.cmd.SetArgs([]string{"testdata/good.yml", "testdata/invalid.yml"})
	require.Error(t, cmd.cmd.Execute())
	require.Equal(t, 2, cmd.checked)
}

func TestCheckConfigThatDoesNotExist(t *testing.T) {
	cmd := newCheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/nope.yml"})
	require.ErrorIs(t, cmd.cmd.Execute(), os.ErrNotExist)
	require.Equal(t, 0, cmd.checked)
}

func TestCheckConfigUnmarshalError(t *testing.T) {
	cmd := newCheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/unmarshal_error.yml"})
	require.EqualError(t, cmd.cmd.Execute(), "yaml: unmarshal errors:\n  line 1: field foo not found in type config.Project")
	require.Equal(t, 0, cmd.checked)
}

func TestCheckConfigInvalid(t *testing.T) {
	cmd := newCheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/invalid.yml"})
	require.Error(t, cmd.cmd.Execute())
	require.Equal(t, 1, cmd.checked)
}

func TestCheckConfigInvalidQuiet(t *testing.T) {
	cmd := newCheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/invalid.yml", "-q"})
	require.Error(t, cmd.cmd.Execute())
	require.Equal(t, 1, cmd.checked)
}

func TestCheckConfigDeprecated(t *testing.T) {
	cmd := newCheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/good.yml", "--deprecated"})
	require.Error(t, cmd.cmd.Execute())
	require.Equal(t, 1, cmd.checked)
}
