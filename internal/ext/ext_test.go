package ext

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/buildtarget"
	"github.com/stretchr/testify/assert"
)

func TestExtWindows(t *testing.T) {
	assert.Equal(t, ".exe", For(buildtarget.New("windows", "", "")))
	assert.Equal(t, ".exe", For(buildtarget.New("windows", "adm64", "")))
}

func TestExtOthers(t *testing.T) {
	assert.Empty(t, "", For(buildtarget.New("linux", "", "")))
	assert.Empty(t, "", For(buildtarget.New("linuxwin", "", "")))
	assert.Empty(t, "", For(buildtarget.New("winasdasd", "sad", "6")))
}
