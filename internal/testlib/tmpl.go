package testlib

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func RequireTemplateError(tb testing.TB, err error) {
	tb.Helper()

	require.Error(tb, err)
	require.Contains(tb, err.Error(), "template:")
	require.Regexp(tb, "bad character|map has no entry|in operand", err.Error())
}
