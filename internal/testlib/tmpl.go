package testlib

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
)

// RequireTemplateError requires thqt an error happens and that it is a template error.
func RequireTemplateError(tb testing.TB, err error) {
	tb.Helper()

	require.Error(tb, err)
	require.ErrorAs(tb, err, &template.ExecError{})
}
