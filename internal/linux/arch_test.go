package linux

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArch(t *testing.T) {
	for from, to := range map[string]string{
		"linuxamd64":   "amd64",
		"linux386":     "i386",
		"linuxarm64":   "arm64",
		"linuxarm5":    "armel",
		"linuxarm6":    "armhf",
		"linuxarm7":    "armhf",
		"linuxppc64":   "ppc64",
		"linuxppc64le": "ppc64le",
		"linuxwhat":    "what",
	} {
		t.Run(fmt.Sprintf("%s to %s", from, to), func(t *testing.T) {
			assert.Equal(t, to, Arch(from))
		})
	}
}
