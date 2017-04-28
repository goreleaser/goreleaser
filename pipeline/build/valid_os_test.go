package build

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValid(t *testing.T) {
	var targets = []buildTarget{
		buildTarget{"android", "arm", ""},
		buildTarget{"darwin", "386", ""},
		buildTarget{"darwin", "amd64", ""},
		buildTarget{"dragonfly", "amd64", ""},
		buildTarget{"freebsd", "386", ""},
		buildTarget{"freebsd", "amd64", ""},
		buildTarget{"freebsd", "arm", "5"},
		buildTarget{"freebsd", "arm", "6"},
		buildTarget{"freebsd", "arm", "7"},
		buildTarget{"linux", "386", ""},
		buildTarget{"linux", "amd64", ""},
		buildTarget{"linux", "arm", "5"},
		buildTarget{"linux", "arm", "6"},
		buildTarget{"linux", "arm", "7"},
		buildTarget{"linux", "arm64", ""},
		buildTarget{"linux", "mips", ""},
		buildTarget{"linux", "mipsle", ""},
		buildTarget{"linux", "mips64", ""},
		buildTarget{"linux", "mips64le", ""},
		buildTarget{"netbsd", "386", ""},
		buildTarget{"netbsd", "amd64", ""},
		buildTarget{"netbsd", "arm", "5"},
		buildTarget{"netbsd", "arm", "6"},
		buildTarget{"netbsd", "arm", "7"},
		buildTarget{"openbsd", "386", ""},
		buildTarget{"openbsd", "amd64", ""},
		buildTarget{"plan9", "386", ""},
		buildTarget{"plan9", "amd64", ""},
		buildTarget{"solaris", "amd64", ""},
		buildTarget{"windows", "386", ""},
		buildTarget{"windows", "amd64", ""},
	}
	for _, target := range targets {
		t.Run(fmt.Sprintf("%v is valid", target.String()), func(t *testing.T) {
			assert.True(t, isValid(target))
		})
	}
}

func TestInvalid(t *testing.T) {
	var targets = []buildTarget{
		buildTarget{"darwin", "arm", ""},
		buildTarget{"darwin", "arm64", ""},
		buildTarget{"windows", "arm", ""},
		buildTarget{"windows", "arm64", ""},
		buildTarget{"linux", "ppc64", ""},
		buildTarget{"linux", "ppc64le", ""},
		buildTarget{"openbsd", "arm", ""},
		buildTarget{"freebsd", "arm", ""},
		buildTarget{"linux", "arm", ""},
		buildTarget{"netbsd", "arm", ""},
	}
	for _, target := range targets {
		t.Run(fmt.Sprintf("%v is invalid", target.String()), func(t *testing.T) {
			assert.False(t, isValid(target))
		})
	}
}
