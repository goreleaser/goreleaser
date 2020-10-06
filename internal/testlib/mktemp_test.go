package testlib

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMkTemp(t *testing.T) {
	current, err := os.Getwd()
	require.NoError(t, err)
	folder, back := Mktmp(t)
	require.NotEmpty(t, folder)
	back()
	newCurrent, err := os.Getwd()
	require.NoError(t, err)
	require.Equal(t, current, newCurrent)
}
