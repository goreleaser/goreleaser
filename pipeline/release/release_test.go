package release

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestDescription(t *testing.T) {
	assert := assert.New(t)
	desc := description("0abf342 some message")
	assert.Contains(desc, "0abf342 some message")
	assert.Contains(desc, "Automated with @goreleaser")
	assert.Contains(desc, "go version go1.")
	fmt.Println(desc)
}
