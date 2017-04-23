package goreleaserlib

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	_ = os.Unsetenv("GITHUB_TOKEN")
}

func TestRelease(t *testing.T) {
	var assert = assert.New(t)
	_, back := setup(t)
	defer back()
	var flags = fakeFlags{
		flags: map[string]string{
			"skip-publish":  "true",
			"skip-validate": "true",
		},
	}
	assert.NoError(Release(flags))
}

func TestConfigFileIsSetAndDontExist(t *testing.T) {
	var assert = assert.New(t)
	var flags = fakeFlags{
		flags: map[string]string{
			"config": "/this/wont/exist",
		},
	}
	assert.Error(Release(flags))
}

func TestReleaseNotesFileDontExist(t *testing.T) {
	var assert = assert.New(t)
	var flags = fakeFlags{
		flags: map[string]string{
			"release-notes": "/this/also/wont/exist",
		},
	}
	assert.Error(Release(flags))
}

func TestCustomReleaseNotesFile(t *testing.T) {
	var assert = assert.New(t)
	folder, back := setup(t)
	defer back()
	var releaseNotes = filepath.Join(folder, "notes.md")
	createFile(t, releaseNotes, "nothing important at all")
	var flags = fakeFlags{
		flags: map[string]string{
			"release-notes": releaseNotes,
			"skip-publish":  "true",
			"skip-validate": "true",
		},
	}
	assert.NoError(Release(flags))
}

func TestBrokenPipe(t *testing.T) {
	var assert = assert.New(t)
	_, back := setup(t)
	defer back()
	createFile(t, "main.go", "not a valid go file")
	var flags = fakeFlags{
		flags: map[string]string{
			"skip-publish":  "true",
			"skip-validate": "true",
		},
	}
	assert.Error(Release(flags))
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
func (f fakeFlags) Bool(s string) bool {
	return f.flags[s] == "true"
}

func setup(t *testing.T) (current string, back func()) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "goreleaser")
	assert.NoError(err)
	log.Println("Folder:", folder)
	previous, err := os.Getwd()
	assert.NoError(err)
	assert.NoError(os.Chdir(folder))
	var gitCmds = [][]string{
		{"init"},
		{"add", "-A"},
		{"commit", "--allow-empty", "-m", "asdf"},
		{"tag", "v0.0.1"},
		{"commit", "--allow-empty", "-m", "asas89d"},
		{"commit", "--allow-empty", "-m", "assssf"},
		{"commit", "--allow-empty", "-m", "assd"},
		{"tag", "v0.0.2"},
		{"remote", "add", "origin", "git@github.com:goreleaser/fake.git"},
	}
	createGoreleaserYaml(t)
	createMainGo(t)
	for _, cmd := range gitCmds {
		var args = []string{
			"-c",
			"user.name='GoReleaser'",
			"-c",
			"user.email='test@goreleaser.github.com'",
		}
		args = append(args, cmd...)
		assert.NoError(exec.Command("git", args...).Run())
	}
	return folder, func() {
		assert.NoError(os.Chdir(previous))
	}
}

func createFile(t *testing.T, filename, contents string) {
	var assert = assert.New(t)
	assert.NoError(ioutil.WriteFile(filename, []byte(contents), 0644))
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
