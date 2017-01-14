package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCurrentTag(t *testing.T) {
	assert := assert.New(t)
	tag, err := currentTag()
	assert.NoError(err)
	assert.NotEmpty(tag)
}

func TestPreviousTag(t *testing.T) {
	assert := assert.New(t)
	tag, err := previousTag("v0.2.0")
	assert.NoError(err)
	assert.NotEmpty(tag)
}

func TestInvalidRef(t *testing.T) {
	assert := assert.New(t)
	tag, err := previousTag("this-should-not-exist")
	assert.Error(err)
	assert.Empty(tag)
}
