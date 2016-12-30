package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixConfig(t *testing.T) {
	assert := assert.New(t)
	config := fix(ProjectConfig{})

	assert.Equal("main.go", config.Build.Main)
	assert.Contains(config.Build.Oses, "darwin")
	assert.Contains(config.Build.Oses, "linux")
	assert.Contains(config.Build.Arches, "386")
	assert.Contains(config.Build.Arches, "amd64")
}

func TestFixConfigMissingFiles(t *testing.T) {
	assert := assert.New(t)
	config := fix(ProjectConfig{})

	assert.NotContains(config.Files, "README.md")
	assert.NotContains(config.Files, "LICENSE.md")
	assert.NotContains(config.Files, "LICENCE.md")
}

func TestFixConfigNoMissingFiles(t *testing.T) {
	assert := assert.New(t)

	os.Chdir("./.test")
	config := fix(ProjectConfig{})

	assert.Contains(config.Files, "README.md")
	assert.Contains(config.Files, "LICENSE.md")
	assert.Contains(config.Files, "LICENCE.md")
}
