package pipe

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSkipPipe(t *testing.T) {
	reason := "this is a test"
	err := Skip(reason)
	require.EqualError(t, err, reason)
}

func TestSkipf(t *testing.T) {
	err := Skipf("foo %s", "bar")
	require.True(t, IsSkip(err))
}

func TestIsSkip(t *testing.T) {
	require.True(t, IsSkip(Skip("whatever")))
	require.False(t, IsSkip(errors.New("nope")))
}

func TestSkipMemento(t *testing.T) {
	m := SkipMemento{}
	m.Remember(Skip("foo"))
	m.Remember(Skip("bar"))
	// test duplicated errors
	m.Remember(Skip("dupe"))
	m.Remember(Skip("dupe"))
	require.EqualError(t, m.Evaluate(), `foo, bar, dupe`)
	require.True(t, IsSkip(m.Evaluate()))
}

func TestSkipMementoNoErrors(t *testing.T) {
	require.NoError(t, (&SkipMemento{}).Evaluate())
}

func TestDetails(t *testing.T) {
	og := errors.New("fake")
	err := NewDetailedError(og, "foo", "bar", "hi", 10)
	require.Equal(t, map[string]any{
		"foo": "bar",
		"hi":  10,
	}, DetailsOf(err))
	require.Empty(t, DetailsOf(og))
	require.ErrorIs(t, err, og)
	require.Equal(t, "fake", err.Error())
}

func TestDetailsOdd(t *testing.T) {
	og := errors.New("fake")
	err := NewDetailedError(og, "foo", "bar", "hi")
	require.Equal(t, map[string]any{
		"foo": "bar",
		"hi":  "missing value",
	}, DetailsOf(err))
	require.Empty(t, DetailsOf(og))
	require.ErrorIs(t, err, og)
	require.Equal(t, "fake", err.Error())
}
