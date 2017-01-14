package build

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtWindows(t *testing.T) {
	assert.Equal(t, extFor("windows"), ".exe")
}

func TestExtOthers(t *testing.T) {
	assert.Empty(t, extFor("linux"))
}
