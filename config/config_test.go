package config

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestFixConfig(t *testing.T) {
	assert := assert.New(t)
	config := fix(ProjectConfig{})
	assert.Equal("releaser", config.BinaryName)
	assert.Equal("main.go", config.Main)
	assert.Contains(config.FileList, "README.md")
	assert.Contains(config.FileList, "LICENSE.md")
	assert.Contains(config.FileList, "releaser")
	assert.Contains(config.Build.Oses, "darwin")
	assert.Contains(config.Build.Oses, "linux")
	assert.Contains(config.Build.Arches, "386")
	assert.Contains(config.Build.Arches, "amd64")
}

func TestFixBadConfig(t *testing.T) {
	assert := assert.New(t)
	config := fix(ProjectConfig{
		FileList: []string{"README.txt"},
	})
	assert.Contains(config.FileList, "releaser")
}