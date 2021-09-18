package logging

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestLogging(t *testing.T) {
	require.NoError(t, Log("foo", func(ctx *context.Context) error {
		return nil
	}, DefaultInitialPadding)(nil))
}
