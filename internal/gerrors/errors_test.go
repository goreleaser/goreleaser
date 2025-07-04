package gerrors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetails(t *testing.T) {
	og := errors.New("fake")
	err := Wrap(og, "message", "foo", "bar", "hi", 10)
	require.Equal(t, map[string]any{
		"foo": "bar",
		"hi":  10,
	}, DetailsOf(err))
	require.Equal(t, 1, ExitOf(err))
	require.Equal(t, 1, ExitOf(og))
	require.Equal(t, "message", MessageOf(err))
	require.Empty(t, DetailsOf(og))
	require.ErrorIs(t, err, og)
	require.Equal(t, "fake", err.Error())
}

func TestDetailsStacking(t *testing.T) {
	og := errors.New("fake")
	err := Wrap(og, "foo", "bar", "hi", 10)
	err = WrapExit(err, "message", 2, "stacked", true)
	require.Equal(t, map[string]any{
		"foo":     "bar",
		"hi":      10,
		"stacked": true,
	}, DetailsOf(err))
	require.Empty(t, DetailsOf(og))
	require.Equal(t, 2, ExitOf(err))
	require.Equal(t, 1, ExitOf(og))
	require.Equal(t, "message", MessageOf(err))
	require.ErrorIs(t, err, og)
	require.Equal(t, "fake", err.Error())
}

func TestDetailsOdd(t *testing.T) {
	og := errors.New("fake")
	err := Wrap(og, "foo", "bar", "hi")
	require.Equal(t, map[string]any{
		"foo": "bar",
		"hi":  "missing value",
	}, DetailsOf(err))
	require.Empty(t, DetailsOf(og))
	require.ErrorIs(t, err, og)
	require.Equal(t, "fake", err.Error())
}
