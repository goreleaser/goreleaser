package git

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCurrentTag(t *testing.T) {
	assert := assert.New(t)
	tag, err := CurrentTag()
	assert.NoError(err)
	assert.NotEmpty(tag)
}

func TestPreviousTag(t *testing.T) {
	assert := assert.New(t)
	tag, err := PreviousTag("v0.2.0")
	assert.NoError(err)
	assert.NotEmpty(tag)
}

func TestInvalidRef(t *testing.T) {
	assert := assert.New(t)
	tag, err := PreviousTag("this-should-not-exist")
	assert.Error(err)
	assert.Empty(tag)
}
