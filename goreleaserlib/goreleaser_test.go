package goreleaserlib

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

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
	var flags = fakeFlags{
		flags: map[string]string{
			"skip-publish":  "true",
			"skip-validate": "true",
			"debug":         "true",
			"parallelism":   "4",
		},
	}
	assert.NoError(t, Release(flags))
}

func TestSnapshotRelease(t *testing.T) {
	_, back := setup(t)
	defer back()
	var flags = fakeFlags{
		flags: map[string]string{
			"snapshot":    "true",
			"parallelism": "4",
		},
	}
	assert.NoError(t, Release(flags))
}

func TestConfigFileIsSetAndDontExist(t *testing.T) {
	var flags = fakeFlags{
		flags: map[string]string{
			"config": "/this/wont/exist",
		},
	}
	assert.Error(t, Release(flags))
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
			var flags = fakeFlags{
				flags: map[string]string{},
			}
			assert.Equal(t, name, getConfigFile(flags))
		})
	}
}

func TestReleaseNotesFileDontExist(t *testing.T) {
	var flags = fakeFlags{
		flags: map[string]string{
			"release-notes": "/this/also/wont/exist",
		},
	}
	assert.Error(t, Release(flags))
}

func TestCustomReleaseNotesFile(t *testing.T) {
	folder, back := setup(t)
	defer back()
	var releaseNotes = filepath.Join(folder, "notes.md")
	createFile(t, releaseNotes, "nothing important at all")
	var flags = fakeFlags{
		flags: map[string]string{
			"release-notes": releaseNotes,
			"skip-publish":  "true",
			"skip-validate": "true",
			"parallelism":   "4",
		},
	}
	assert.NoError(t, Release(flags))
}

func TestBrokenPipe(t *testing.T) {
	_, back := setup(t)
	defer back()
	createFile(t, "main.go", "not a valid go file")
	var flags = fakeFlags{
		flags: map[string]string{
			"skip-publish":  "true",
			"skip-validate": "true",
			"parallelism":   "4",
		},
	}
	assert.Error(t, Release(flags))
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
	flags map[string]string
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
