package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLog(t *testing.T) {
	assert := assert.New(t)
	tag, err := currentTag()
	assert.NoError(err)
	tagb, err := previousTag(tag)
	assert.NoError(err)
	log, err := log(tagb, tag)
	assert.NoError(err)
	assert.NotEmpty(log)
}

func TestLogInvalidRef(t *testing.T) {
	assert := assert.New(t)
	log, err := log("wtfff", "nope")
	assert.Error(err)
	assert.Empty(log)
}
