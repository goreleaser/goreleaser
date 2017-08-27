package linux

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArch(t *testing.T) {
	for from, to := range map[string]string{
		"amd64": "amd64",
		"386":   "i386",
		"arm64": "arm64",
		"arm6":  "armhf",
		"what":  "what",
	} {
		t.Run(fmt.Sprintf("%s to %s", from, to), func(t *testing.T) {
			assert.Equal(t, to, Arch(from))
		})
	}
}
