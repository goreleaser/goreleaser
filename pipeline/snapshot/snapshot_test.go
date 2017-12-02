package snapshot

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringer(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestDefault(t *testing.T) {
	// TODO: implement this
}
