package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	require.NotEmpty(t, buildVersion("test", "test", "test", "test").String())
}
