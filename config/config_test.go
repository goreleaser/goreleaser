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
	assert.Contains(config.Build.GoOS, "darwin")
	assert.Contains(config.Build.GoOS, "linux")
	assert.Contains(config.Build.GoArch, "386")
	assert.Contains(config.Build.GoArch, "amd64")
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
	if err := os.Chdir(dir); err != nil {
		panic(err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			panic(err)
		}
	}()

	config := ProjectConfig{}
	err := config.fillFiles()

	assert.NoError(err)
	assert.Equal(files, config.Files)
}
