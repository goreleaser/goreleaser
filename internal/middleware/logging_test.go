package middleware

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogging(t *testing.T) {
	require.NoError(t, Logging("foo", mockAction(nil), DefaultInitialPadding)(ctx))
}
