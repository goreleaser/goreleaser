package uname

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestUname(t *testing.T) {
	assert := assert.New(t)
	assert.Equal("Darwin", FromGo("darwin"))
	assert.Equal("blah", FromGo("blah"))
}
