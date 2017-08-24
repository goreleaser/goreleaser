package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSkipPipe(t *testing.T) {
	var assert = assert.New(t)
	var reason = "this is a test"
	var err = Skip(reason)
	assert.Error(err)
	assert.Equal(reason, err.Error())
}
