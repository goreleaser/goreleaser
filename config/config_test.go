package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFixConfig(t *testing.T) {
	assert := assert.New(t)
	config := fix(ProjectConfig{})
	assert.Equal("main.go", config.Build.Main)
	assert.Contains(config.Files, "README.md")
	assert.Contains(config.Files, "LICENSE.md")
	assert.Contains(config.Build.Oses, "darwin")
	assert.Contains(config.Build.Oses, "linux")
	assert.Contains(config.Build.Arches, "386")
	assert.Contains(config.Build.Arches, "amd64")
}
