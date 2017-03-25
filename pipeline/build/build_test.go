package build

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestValid(t *testing.T) {
	assert.True(t, valid("windows", "386"))
	assert.True(t, valid("linux", "386"))
	assert.False(t, valid("windows", "arm"))
}
