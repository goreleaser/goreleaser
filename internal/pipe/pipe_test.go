package pipe

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSkipPipe(t *testing.T) {
	var reason = "this is a test"
	var err = Skip(reason)
	require.Error(t, err)
	require.Equal(t, reason, err.Error())
}

func TestIsSkip(t *testing.T) {
	require.True(t, IsSkip(Skip("whatever")))
	require.False(t, IsSkip(errors.New("nope")))
}

func TestSkipMemento(t *testing.T) {
	var m = SkipMemento{}
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
