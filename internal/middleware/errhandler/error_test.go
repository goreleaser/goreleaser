package errhandler

import (
	"fmt"
	"testing"

	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestError(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		require.NoError(t, Handle(func(ctx *context.Context) error {
			return nil
		})(nil))
	})

	t.Run("pipe skipped", func(t *testing.T) {
		require.NoError(t, Handle(func(ctx *context.Context) error {
			return pipe.ErrSkipValidateEnabled
		})(nil))
	})

	t.Run("some err", func(t *testing.T) {
		require.Error(t, Handle(func(ctx *context.Context) error {
			return fmt.Errorf("pipe errored")
		})(nil))
	})
}
