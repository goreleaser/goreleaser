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

	assert.Equal([]string{}, config.Files)
}

func TestFixConfigUSENMarkdown(t *testing.T) {
	assert := assert.New(t)

	cwd, _ := os.Getwd()
	os.Chdir("./.test/1")

	config := fix(ProjectConfig{})
	assert.Equal([]string{"LICENSE.md", "README.md"}, config.Files)

	os.Chdir(cwd)
}

func TestFixConfigRealENMarkdown(t *testing.T) {
	assert := assert.New(t)

	cwd, _ := os.Getwd()
	os.Chdir("./.test/2")

	config := fix(ProjectConfig{})
	assert.Equal([]string{"LICENCE.md", "README.md"}, config.Files)

	os.Chdir(cwd)
}

func TestFixConfigArbitratryENTXT(t *testing.T) {
	assert := assert.New(t)

	cwd, _ := os.Getwd()
	os.Chdir("./.test/3")

	config := fix(ProjectConfig{})
	assert.Equal([]string{"LICENCE.txt", "README.txt"}, config.Files)

	os.Chdir(cwd)
}
