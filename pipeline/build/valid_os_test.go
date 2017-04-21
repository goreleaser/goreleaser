package build

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValid(t *testing.T) {
	var platforms = []struct {
		os, arch string
	}{
		{"android", "arm"},
		{"darwin", "386"},
		{"darwin", "amd64"},
		{"dragonfly", "amd64"},
		{"freebsd", "386"},
		{"freebsd", "amd64"},
		{"freebsd", "arm"},
		{"linux", "386"},
		{"linux", "amd64"},
		{"linux", "arm"},
		{"linux", "arm64"},
		{"linux", "ppc64"},
		{"linux", "ppc64le"},
		{"linux", "mips"},
		{"linux", "mipsle"},
		{"linux", "mips64"},
		{"linux", "mips64le"},
		{"netbsd", "386"},
		{"netbsd", "amd64"},
		{"netbsd", "arm"},
		{"openbsd", "386"},
		{"openbsd", "amd64"},
		{"openbsd", "arm"},
		{"plan9", "386"},
		{"plan9", "amd64"},
		{"solaris", "amd64"},
		{"windows", "386"},
		{"windows", "amd64"},
	}
	for _, p := range platforms {
		t.Run(fmt.Sprintf("%v %v is valid", p.os, p.arch), func(t *testing.T) {
			assert.True(t, valid(p.os, p.arch))
		})
	}
}

func TestInvalid(t *testing.T) {
	var platforms = []struct {
		os, arch string
	}{
		{"darwin", "arm"},
		{"darwin", "arm64"},
		{"windows", "arm"},
		{"windows", "arm64"},
	}
	for _, p := range platforms {
		t.Run(fmt.Sprintf("%v %v is invalid", p.os, p.arch), func(t *testing.T) {
			assert.False(t, valid(p.os, p.arch))
		})
	}
}
