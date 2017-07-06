package buildtarget

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnv(t *testing.T) {
	var assert = assert.New(t)
	assert.Equal(
		[]string{"GOOS=linux", "GOARCH=arm64", "GOARM=6"},
		New("linux", "arm64", "6").Env(),
	)
}

func TestString(t *testing.T) {
	var assert = assert.New(t)
	assert.Equal(
		"linuxarm7",
		New("linux", "arm", "7").String(),
	)
}

func TestPrettyString(t *testing.T) {
	var assert = assert.New(t)
	assert.Equal(
		"linux/arm646",
		New("linux", "arm64", "6").PrettyString(),
	)
}
