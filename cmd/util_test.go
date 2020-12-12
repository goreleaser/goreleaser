package cmd

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/require"
)

type exitMemento struct {
	code int
}

func (e *exitMemento) Exit(i int) {
	e.code = i
}

func setup(t testing.TB) string {
	_ = os.Unsetenv("GITHUB_TOKEN")
	_ = os.Unsetenv("GITLAB_TOKEN")

	previous, err := os.Getwd()
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, os.Chdir(previous))
	})

	var folder = t.TempDir()
	require.NoError(t, os.Chdir(folder))

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

	return folder
}

func createFile(t testing.TB, filename, contents string) {
	require.NoError(t, ioutil.WriteFile(filename, []byte(contents), 0644))
}

func createMainGo(t testing.TB) {
	createFile(t, "main.go", "package main\nfunc main() {println(0)}")
}

func goModInit(t testing.TB) {
	createFile(t, "go.mod", `module foo

go 1.15
`)
}

func createGoreleaserYaml(t testing.TB) {
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
