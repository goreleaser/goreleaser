package testlib

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// RequireTemplateError requires thqt an error happens and that it is a template error.
func RequireTemplateError(tb testing.TB, err error) {
	tb.Helper()

	require.Error(tb, err)
	require.Contains(tb, err.Error(), "template:")
	require.Regexp(tb, "bad character|map has no entry|unexpected \"}\" in operand", err.Error())
}
