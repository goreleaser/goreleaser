package split

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestSplit(t *testing.T) {
	assert := assert.New(t)
	a, b := OnSlash("a/b")
	assert.Equal("a", a)
	assert.Equal("b", b)
}
