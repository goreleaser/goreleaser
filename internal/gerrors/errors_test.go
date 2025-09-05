package gerrors

import (
	"errors"
	"fmt"
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
	err := Wrap(og, "message1", "foo", "bar", "hi", 10)
	err = WrapExit(err, "message2", 2, "stacked", true)
	require.Equal(t, map[string]any{
		"foo":     "bar",
		"hi":      10,
		"stacked": true,
	}, DetailsOf(err))
	require.Empty(t, DetailsOf(og))
	require.Equal(t, 2, ExitOf(err))
	require.Equal(t, 1, ExitOf(og))
	require.Equal(t, "message2", MessageOf(err))
	require.ErrorIs(t, err, og)
	require.Equal(t, "fake", err.Error())
}

func TestDetailsOdd(t *testing.T) {
	og := errors.New("fake")
	err := Wrap(og, "message", "foo", "bar", "hi")
	require.Equal(t, map[string]any{
		"foo": "bar",
		"hi":  "missing value",
	}, DetailsOf(err))
	require.Empty(t, DetailsOf(og))
	require.ErrorIs(t, err, og)
	require.Equal(t, "fake", err.Error())
}

func TestDetailsMultipleWraps(t *testing.T) {
	og := errors.New("fake")
	err := Wrap(og, "hello", "foo", "bar", "zaz", "something")
	err = fmt.Errorf("some more stuff: %w", err)
	err = Wrap(err, "hello one more time", "foo", "again we wrap")
	err = Wrap(err, "hello again", "test2", "another msg")
	require.Equal(t, map[string]any{
		"foo":   "again we wrap",
		"test2": "another msg",
		"zaz":   "something",
	}, DetailsOf(err))
}
