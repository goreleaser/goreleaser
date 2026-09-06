package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/stretchr/testify/require"
)

type exitMemento struct {
	code int
}

func (e *exitMemento) Exit(i int) {
	e.code = i
}

// mktmp makes an empty directory the working directory of the test.
//
// Use it instead of [setup] when the test does not run a pipeline: setup also
// creates a buildable Go module and a git repository with four commits and two
// tags, which costs nine git processes.
func mktmp(tb testing.TB) string {
	tb.Helper()

	_ = os.Unsetenv("GITHUB_TOKEN")
	_ = os.Unsetenv("GITLAB_TOKEN")
	_ = os.Unsetenv("GITEA_TOKEN")

	folder := tb.TempDir()
	tb.Chdir(folder)
	return folder
}

func setup(tb testing.TB) {
	tb.Helper()

	mktmp(tb)

	createGoReleaserYaml(tb)
	createMainGo(tb)
	goModInit(tb)
	testlib.GitInit(tb)
	testlib.GitAdd(tb)
	testlib.GitCommit(tb, "asdf")
	testlib.GitTag(tb, "v0.0.1")
	testlib.GitCommit(tb, "asas89d")
	testlib.GitCommit(tb, "assssf")
	testlib.GitCommit(tb, "assd")
	testlib.GitTag(tb, "v0.0.2")
	testlib.GitRemoteAdd(tb, "git@github.com:goreleaser/fake.git")
}

func createFile(tb testing.TB, filename, contents string) {
	tb.Helper()
	require.NoError(tb, os.WriteFile(filename, []byte(contents), 0o644))
}

func createMainGo(tb testing.TB) {
	tb.Helper()
	createFile(tb, "main.go", "package main\nfunc main() {println(0)}")
}

func goModInit(tb testing.TB) {
	tb.Helper()
	createFile(tb, "go.mod", fmt.Sprintf(`module foo

go %s
`, testlib.GoVersion))
}

func createGoReleaserYaml(tb testing.TB) {
	tb.Helper()
	yaml := `builds:
- binary: 'fake{{if .IsSnapshot}}_snapshot{{end}}'
  goos:
    - linux
  goarch:
    - amd64
release:
  github:
    owner: goreleaser
    name: fake
`
	createFile(tb, "goreleaser.yml", yaml)
}

func setupPro(tb testing.TB) {
	tb.Helper()

	mktmp(tb)

	createProGoReleaserYaml(tb)
	createMainGo(tb)
	goModInit(tb)
	testlib.GitInit(tb)
	testlib.GitAdd(tb)
	testlib.GitCommit(tb, "asdf")
	testlib.GitTag(tb, "v0.0.1")
	testlib.GitCommit(tb, "asas89d")
	testlib.GitCommit(tb, "assssf")
	testlib.GitCommit(tb, "assd")
	testlib.GitTag(tb, "v0.0.2")
	testlib.GitRemoteAdd(tb, "git@github.com:goreleaser/fake.git")
}

func createProGoReleaserYaml(tb testing.TB) {
	tb.Helper()
	yaml := `version: 2
pro: true
includes: []
builds:
- binary: 'fake{{if .IsSnapshot}}_snapshot{{end}}'
  goos:
    - linux
  goarch:
    - amd64
release:
  github:
    owner: goreleaser
    name: fake
`
	createFile(tb, "goreleaser.yml", yaml)
}
