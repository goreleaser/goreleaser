package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootCmd(t *testing.T) {
	var cmd = NewRootCmd("foo")

	require.NoError(t, cmd.Execute())
}
