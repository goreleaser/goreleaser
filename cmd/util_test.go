package cmd

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
)

type exitMemento struct {
	code int
}

func (e *exitMemento) Exit(i int) {
	e.code = i
}

func setup(t *testing.T) (current string, back func()) {
	_ = os.Unsetenv("GITHUB_TOKEN")
	_ = os.Unsetenv("GITLAB_TOKEN")

	folder, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	previous, err := os.Getwd()
	assert.NoError(t, err)
	assert.NoError(t, os.Chdir(folder))
	createGoreleaserYaml(t)
	createMainGo(t)
	goModInit(t)
	testlib.GitInit(t)
	testlib.GitAdd(t)
	testlib.GitCommit(t, "asdf")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "asas89d")
	testlib.GitCommit(t, "assssf")
	testlib.GitCommit(t, "assd")
	testlib.GitTag(t, "v0.0.2")
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/fake.git")
	return folder, func() {
		assert.NoError(t, os.Chdir(previous))
	}
}

func createFile(t *testing.T, filename, contents string) {
	assert.NoError(t, ioutil.WriteFile(filename, []byte(contents), 0644))
}

func createMainGo(t *testing.T) {
	createFile(t, "main.go", "package main\nfunc main() {println(0)}")
}

func goModInit(t *testing.T) {
	createFile(t, "go.mod", `module foo

go 1.14
`)
}

func createGoreleaserYaml(t *testing.T) {
	var yaml = `build:
  binary: fake
  goos:
    - linux
  goarch:
    - amd64
release:
  github:
    owner: goreleaser
    name: fake
`
	createFile(t, "goreleaser.yml", yaml)
}
