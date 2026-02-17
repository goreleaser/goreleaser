package gerrors

import (
	"errors"
	"maps"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetails(t *testing.T) {
	og := errors.New("fake")
	err := Wrap(
		og,
		WithMessage("message"),
		WithDetails(
			"foo", "bar",
			"hi", 10,
		),
	)

	de, ok := errors.AsType[ErrDetailed](err)
	require.True(t, ok)

	require.Equal(t, map[string]any{
		"foo": "bar",
		"hi":  10,
	}, maps.Collect(de.Details()))
	require.Equal(t, 1, de.Exit())
	require.Equal(t, []string{"message"}, de.Messages())
	require.ErrorIs(t, err, og)
	require.Equal(t, "fake", err.Error())
}

func TestDetailsStacking(t *testing.T) {
	og := errors.New("fake")
	err := Wrap(
		og,
		WithMessage("message1"),
		WithDetails(
			"foo", "bar",
			"hi", 10,
		),
	)
	err = Wrap(
		err,
		WithMessage("message2"),
		WithExit(2),
		WithDetails("stacked", true),
	)

	de, ok := errors.AsType[ErrDetailed](err)
	require.True(t, ok)

	require.Equal(t, map[string]any{
		"foo":     "bar",
		"hi":      10,
		"stacked": true,
	}, maps.Collect(de.Details()))
	require.Equal(t, 2, de.Exit())
	require.Equal(t, []string{"message2", "message1"}, de.Messages())
	require.Equal(t, "fake", err.Error())
}

func TestDetailsOdd(t *testing.T) {
	og := errors.New("fake")
	err := Wrap(
		og,
		WithMessage("message"),
		WithDetails("foo", "bar", "hi"),
	)

	de, ok := errors.AsType[ErrDetailed](err)
	require.True(t, ok)

	require.Equal(t, map[string]any{
		"foo": "bar",
		"hi":  "missing value",
	}, maps.Collect(de.Details()))
	require.ErrorIs(t, err, og)
	require.Equal(t, "fake", err.Error())
}
