package ext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtWindows(t *testing.T) {
	assert.Equal(t, ".exe", For("windows"))
	assert.Equal(t, ".exe", For("windowsamd64"))
}

func TestExtOthers(t *testing.T) {
	assert.Empty(t, "", For("linux"))
	assert.Empty(t, "", For("linuxwin"))
	assert.Empty(t, "", For("winasdasd"))
}
