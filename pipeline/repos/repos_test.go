package repos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplit(t *testing.T) {
	assert := assert.New(t)
	a, b := split("a/b")
	assert.Equal("a", a)
	assert.Equal("b", b)
}
