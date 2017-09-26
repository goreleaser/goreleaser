package pipeline

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
