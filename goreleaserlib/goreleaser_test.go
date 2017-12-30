package goreleaserlib

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func init() {
	_ = os.Unsetenv("GITHUB_TOKEN")
}

func TestRelease(t *testing.T) {
	_, back := setup(t)
	defer back()
	assert.NoError(t, Release(newFlags(t, testParams())))
}

func TestSnapshotRelease(t *testing.T) {
	_, back := setup(t)
	defer back()
	params := testParams()
	params["snapshot"] = "true"
	assert.NoError(t, Release(newFlags(t, params)))
}

func TestConfigFileIsSetAndDontExist(t *testing.T) {
	params := testParams()
	params["config"] = "/this/wont/exist"
	assert.Error(t, Release(newFlags(t, params)))
}

func TestConfigFlagNotSetButExists(t *testing.T) {
	for _, name := range []string{
		".goreleaser.yml",
		".goreleaser.yaml",
		"goreleaser.yml",
		"goreleaser.yaml",
	} {
		t.Run(name, func(t *testing.T) {
			folder, back := setup(t)
			defer back()
			assert.NoError(
				t,
				os.Rename(
					filepath.Join(folder, "goreleaser.yml"),
					filepath.Join(folder, name),
				),
			)
			assert.Equal(t, name, getConfigFile(newFlags(t, testParams())))
		})
	}
}

func TestReleaseNotesFileDontExist(t *testing.T) {
	params := testParams()
	params["release-notes"] = "/this/also/wont/exist"
	assert.Error(t, Release(newFlags(t, params)))
}

func TestCustomReleaseNotesFile(t *testing.T) {
	folder, back := setup(t)
	defer back()
	var releaseNotes = filepath.Join(folder, "notes.md")
	createFile(t, releaseNotes, "nothing important at all")
	var params = testParams()
	params["release-notes"] = releaseNotes
	assert.NoError(t, Release(newFlags(t, params)))
}

func TestBrokenPipe(t *testing.T) {
	_, back := setup(t)
	defer back()
	createFile(t, "main.go", "not a valid go file")
	assert.Error(t, Release(newFlags(t, testParams())))
}

func TestInitProject(t *testing.T) {
	_, back := setup(t)
	defer back()
	var filename = "test_goreleaser.yml"
	assert.NoError(t, InitProject(filename))

	file, err := os.Open(filename)
	assert.NoError(t, err)
	out, err := ioutil.ReadAll(file)
	assert.NoError(t, err)

	var config = config.Project{}
	assert.NoError(t, yaml.Unmarshal(out, &config))
}

func TestInitProjectFileExist(t *testing.T) {
	_, back := setup(t)
	defer back()
	var filename = "test_goreleaser.yml"
	createFile(t, filename, "")
	assert.Error(t, InitProject(filename))
}

func TestInitProjectDefaultPipeFails(t *testing.T) {
	_, back := setup(t)
	defer back()
	var filename = "test_goreleaser.yml"
	assert.NoError(t, os.RemoveAll(".git"))
	assert.Error(t, InitProject(filename))
}

// fakeFlags is a mock of the cli flags
type fakeFlags struct {
	t     *testing.T
	flags map[string]string
}

func newFlags(t *testing.T, params map[string]string) Flags {
	return fakeFlags{
		t:     t,
		flags: params,
	}
}

func (f fakeFlags) IsSet(s string) bool {
	return f.flags[s] != ""
}

func (f fakeFlags) String(s string) string {
	return f.flags[s]
}

func (f fakeFlags) Int(s string) int {
	i, _ := strconv.ParseInt(f.flags[s], 10, 32)
	return int(i)
}

func (f fakeFlags) Bool(s string) bool {
	return f.flags[s] == "true"
}

func (f fakeFlags) Duration(s string) time.Duration {
	result, err := time.ParseDuration(f.flags[s])
	assert.NoError(f.t, err)
	return result
}

func testParams() map[string]string {
	return map[string]string{
		"debug":         "true",
		"parallelism":   "4",
		"skip-publish":  "true",
		"skip-validate": "true",
		"timeout":       "1m",
	}
}

func setup(t *testing.T) (current string, back func()) {
	folder, err := ioutil.TempDir("", "goreleaser")
	assert.NoError(t, err)
	previous, err := os.Getwd()
	assert.NoError(t, err)
	assert.NoError(t, os.Chdir(folder))
	createGoreleaserYaml(t)
	createMainGo(t)
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
