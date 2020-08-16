package pipe

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSkipPipe(t *testing.T) {
	var reason = "this is a test"
	var err = Skip(reason)
	assert.Error(t, err)
	assert.Equal(t, reason, err.Error())
}

func TestIsSkip(t *testing.T) {
	assert.True(t, IsSkip(Skip("whatever")))
	assert.False(t, IsSkip(errors.New("nope")))
}

func TestSkipMemento(t *testing.T) {
	var m = SkipMemento{}
	m.Remember(Skip("foo"))
	m.Remember(Skip("bar"))
	// test duplicated errors
	m.Remember(Skip("dupe"))
	m.Remember(Skip("dupe"))
	assert.EqualError(t, m.Evaluate(), `foo, bar, dupe`)
	assert.True(t, IsSkip(m.Evaluate()))
}

func TestSkipMementoNoErrors(t *testing.T) {
	assert.NoError(t, (&SkipMemento{}).Evaluate())
}
