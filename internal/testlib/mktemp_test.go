package testlib

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMkTemp(t *testing.T) {
	require.NotEmpty(t, Mktmp(t))
}
