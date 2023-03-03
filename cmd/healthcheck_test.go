package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHealthcheckSystem(t *testing.T) {
	cmd := newHealthcheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/good.yml"})
	require.NoError(t, cmd.cmd.Execute())
}

func TestHealthcheckConfigThatDoesNotExist(t *testing.T) {
	cmd := newHealthcheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/nope.yml"})
	require.EqualError(t, cmd.cmd.Execute(), "open testdata/nope.yml: no such file or directory")
}

func TestHealthcheckMissingTool(t *testing.T) {
	cmd := newHealthcheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/missing_tool.yml"})
	require.EqualError(t, cmd.cmd.Execute(), "one or more needed tools are not present")
}

func TestHealthcheckQuier(t *testing.T) {
	cmd := newHealthcheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/good.yml", "--quiet"})
	require.NoError(t, cmd.cmd.Execute())
}
