package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert := assert.New(t)
	desc := describeRelease("0abf342 some message")
	assert.Contains(desc, "0abf342 some message")
	assert.Contains(desc, "Automated with @goreleaser")
	assert.Contains(desc, "go version go1.")
}
