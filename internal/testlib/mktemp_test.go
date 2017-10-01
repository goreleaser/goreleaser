package testlib

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMkTemp(t *testing.T) {
	current, err := os.Getwd()
	assert.NoError(t, err)
	folder, back := Mktmp(t)
	assert.NotEmpty(t, folder)
	back()
	newCurrent, err := os.Getwd()
	assert.NoError(t, err)
	assert.Equal(t, current, newCurrent)
}
