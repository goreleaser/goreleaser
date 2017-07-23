package testlib

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMkTemp(t *testing.T) {
	var assert = assert.New(t)
	current, err := os.Getwd()
	assert.NoError(err)
	folder, back := Mktmp(t)
	assert.NotEmpty(folder)
	back()
	newCurrent, err := os.Getwd()
	assert.NoError(err)
	assert.Equal(current, newCurrent)
}
