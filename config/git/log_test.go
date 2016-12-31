package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLog(t *testing.T) {
	assert := assert.New(t)
	log, err := Log("v0.1.9", "v0.2.0")
	assert.NoError(err)
	assert.NotEmpty(log)
}

func TestLogInvalidRef(t *testing.T) {
	assert := assert.New(t)
	log, err := Log("wtfff", "nope")
	assert.Error(err)
	assert.Empty(log)
}
