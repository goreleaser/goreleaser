package skip

import (
	"fmt"
	"testing"

	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestSkip(t *testing.T) {
	fakeErr := fmt.Errorf("fake error")
	action := func(_ *context.Context) error {
		return fakeErr
	}

	t.Run("not a skipper", func(t *testing.T) {
		require.EqualError(t, Maybe(action, action)(nil), fakeErr.Error())
	})

	t.Run("skip", func(t *testing.T) {
		require.NoError(t, Maybe(skipper{true}, action)(nil))
	})

	t.Run("do not skip", func(t *testing.T) {
		require.EqualError(t, Maybe(skipper{false}, action)(nil), fakeErr.Error())
	})
}

func TestSkipErr(t *testing.T) {
	fakeErr := fmt.Errorf("fake error")
	action := func(_ *context.Context) error {
		return fakeErr
	}

	t.Run("no err", func(t *testing.T) {
		require.NoError(t, Maybe(errSkipper{true, nil}, action)(nil))
	})

	t.Run("with err", func(t *testing.T) {
		require.EqualError(t, Maybe(
			errSkipper{false, fmt.Errorf("skip err")},
			action,
		)(nil), "skip blah: skip err")
	})
}

var (
	_ Skipper    = skipper{}
	_ ErrSkipper = errSkipper{}
)

type skipper struct {
	skip bool
}

func (s skipper) String() string { return "blah" }

func (s skipper) Skip(_ *context.Context) bool {
	return s.skip
}

type errSkipper struct {
	skip bool
	err  error
}

func (s errSkipper) String() string { return "blah" }

func (s errSkipper) Skip(_ *context.Context) (bool, error) {
	return s.skip, s.err
}
