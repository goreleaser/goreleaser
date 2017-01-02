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
	assertFiles(t, "./.test/1", []string{"LICENSE.md", "README.md"})
}

func TestFillFilesRealENMarkdown(t *testing.T) {
	assertFiles(t, "./.test/2", []string{"LICENCE.md", "README.md"})
}

func TestFillFilesArbitratryENTXT(t *testing.T) {
	assertFiles(t, "./.test/3", []string{"LICENCE.txt", "README.txt"})
}

func TestFillFilesArbitratryENNoSuffix(t *testing.T) {
	assertFiles(t, "./.test/4", []string{"LICENCE"})
}

func TestFillFilesChangelog(t *testing.T) {
	assertFiles(t, "./.test/5", []string{"CHANGELOG", "CHANGELOG.md"})
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

func assertFiles(t *testing.T, dir string, files []string) {
	assert := assert.New(t)

	cwd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(cwd)

	config := ProjectConfig{}
	err := config.fillFiles()

	assert.NoError(err)
	assert.Equal(files, config.Files)
}
