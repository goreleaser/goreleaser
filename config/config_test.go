package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFillBasicData(t *testing.T) {
	assert := assert.New(t)
	config := ProjectConfig{}
	config.fillBasicData()

	assert.Equal("main.go", config.Build.Main)
	assert.Contains(config.Build.Oses, "darwin")
	assert.Contains(config.Build.Oses, "linux")
	assert.Contains(config.Build.Arches, "386")
	assert.Contains(config.Build.Arches, "amd64")
}

func TestFillFilesMissingFiles(t *testing.T) {
	assert := assert.New(t)
	config := ProjectConfig{}
	err := config.fillFiles()

	assert.NoError(err)
	assert.Equal([]string{}, config.Files)
}

func TestFillFilesUSENMarkdown(t *testing.T) {
	assert := assert.New(t)

	cwd, _ := os.Getwd()
	os.Chdir("./.test/1")
	defer os.Chdir(cwd)

	config := ProjectConfig{}
	err := config.fillFiles()

	assert.NoError(err)
	assert.Equal([]string{"LICENSE.md", "README.md"}, config.Files)
}

func TestFillFilesRealENMarkdown(t *testing.T) {
	assert := assert.New(t)

	cwd, _ := os.Getwd()
	os.Chdir("./.test/2")
	defer os.Chdir(cwd)

	config := ProjectConfig{}
	err := config.fillFiles()

	assert.NoError(err)
	assert.Equal([]string{"LICENCE.md", "README.md"}, config.Files)
}

func TestFillFilesArbitratryENTXT(t *testing.T) {
	assert := assert.New(t)

	cwd, _ := os.Getwd()
	os.Chdir("./.test/3")
	defer os.Chdir(cwd)

	config := ProjectConfig{}
	err := config.fillFiles()

	assert.NoError(err)
	assert.Equal([]string{"LICENCE.txt", "README.txt"}, config.Files)
}

func TestFillFilesArbitratryENNoSuffix(t *testing.T) {
	assert := assert.New(t)

	cwd, _ := os.Getwd()
	os.Chdir("./.test/4")
	defer os.Chdir(cwd)

	config := ProjectConfig{}
	err := config.fillFiles()

	assert.NoError(err)
	assert.Equal([]string{"LICENCE"}, config.Files)
}

func TestValidadeMissingBinaryName(t *testing.T) {
	assert := assert.New(t)

	config := ProjectConfig{Repo: "asd/asd"}
	assert.Error(config.validate())
}

func TestValidadeMissingRepo(t *testing.T) {
	assert := assert.New(t)

	config := ProjectConfig{BinaryName: "asd"}
	assert.Error(config.validate())
}

func TestValidadeMinimalConfig(t *testing.T) {
	assert := assert.New(t)

	config := ProjectConfig{BinaryName: "asd", Repo: "asd/asd"}
	assert.NoError(config.validate())
}
