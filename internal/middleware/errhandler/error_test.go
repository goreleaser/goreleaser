package errhandler

import (
	"errors"
	"fmt"
	"testing"

	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/hashicorp/go-multierror"
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

func TestErrorMemo(t *testing.T) {
	memo := Memo{}
	t.Run("no errors", func(t *testing.T) {
		require.NoError(t, memo.Wrap(func(ctx *context.Context) error {
			return nil
		})(nil))
	})

	t.Run("pipe skipped", func(t *testing.T) {
		require.NoError(t, memo.Wrap(func(ctx *context.Context) error {
			return pipe.ErrSkipValidateEnabled
		})(nil))
	})

	t.Run("some err", func(t *testing.T) {
		require.NoError(t, memo.Wrap(func(ctx *context.Context) error {
			return fmt.Errorf("pipe errored")
		})(nil))
	})

	err := memo.Error()
	merr := &multierror.Error{}
	require.True(t, errors.As(err, &merr), "must be a multierror")
	require.Len(t, merr.Errors, 1)
}
