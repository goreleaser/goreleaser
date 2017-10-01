package buildtarget

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnv(t *testing.T) {
	assert.Equal(
		t,
		[]string{"GOOS=linux", "GOARCH=arm64", "GOARM=6"},
		New("linux", "arm64", "6").Env(),
	)
}

func TestString(t *testing.T) {
	assert.Equal(
		t,
		"linuxarm7",
		New("linux", "arm", "7").String(),
	)
}

func TestPrettyString(t *testing.T) {
	assert.Equal(
		t,
		"linux/arm646",
		New("linux", "arm64", "6").PrettyString(),
	)
}
