package compress

import (
	"testing"
	"github.com/docker/docker/pkg/testutil/assert"
)

func TestExtWindows(t *testing.T) {
	assert.Equal(t, ext("windows"), ".exe")
}

func TestExtOthers(t *testing.T) {
	assert.Equal(t, ext("linux"), "")
}
