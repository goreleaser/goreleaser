package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
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

func TestReleaseProject(t *testing.T) {
	_, back := setup(t)
	defer back()
	assert.NoError(t, releaseProject(testParams()))
}

func TestReleaseProjectSkipPublish(t *testing.T) {
	_, back := setup(t)
	defer back()
	params := testParams()
	params.Snapshot = true
	params.SkipPublish = true
	assert.NoError(t, releaseProject(params))
}

func TestConfigFileIsSetAndDontExist(t *testing.T) {
	params := testParams()
	params.Config = "/this/wont/exist"
	assert.Error(t, releaseProject(params))
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
			err := os.Rename(
				filepath.Join(folder, "goreleaser.yml"),
				filepath.Join(folder, name),
			)
			assert.NoError(t, err)
			proj, err := loadConfig("")
			assert.NoError(t, err)
			assert.NotEqual(t, config.Project{}, proj)
		})
	}
}

func TestConfigFileDoesntExist(t *testing.T) {
	folder, back := setup(t)
	defer back()
	err := os.Remove(filepath.Join(folder, "goreleaser.yml"))
	assert.NoError(t, err)
	proj, err := loadConfig("")
	assert.NoError(t, err)
	assert.Equal(t, config.Project{}, proj)
}

func TestReleaseNotesFileDontExist(t *testing.T) {
	params := testParams()
	params.ReleaseNotes = "/this/also/wont/exist"
	assert.Error(t, releaseProject(params))
}

func TestCustomReleaseNotesFile(t *testing.T) {
	_, back := setup(t)
	defer back()
	releaseNotes, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	createFile(t, releaseNotes.Name(), "nothing important at all")
	var params = testParams()
	params.ReleaseNotes = releaseNotes.Name()
	assert.NoError(t, releaseProject(params))
}

func TestBrokenPipe(t *testing.T) {
	_, back := setup(t)
	defer back()
	createFile(t, "main.go", "not a valid go file")
	assert.Error(t, releaseProject(testParams()))
}

func TestInitProject(t *testing.T) {
	_, back := setup(t)
	defer back()
	var filename = "test_goreleaser.yml"
	assert.NoError(t, initProject(filename))

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
	assert.Error(t, initProject(filename))
}

func TestInitProjectDefaultPipeFails(t *testing.T) {
	folder, back := setup(t)
	defer back()
	var filename = "test_goreleaser.yml"
	assert.NoError(t, os.Chmod(folder, 0000))
	assert.EqualError(t, initProject(filename), `stat test_goreleaser.yml: permission denied`)
}

func testParams() releaseOptions {
	return releaseOptions{
		Debug:       true,
		Parallelism: 4,
		Snapshot:    true,
		Timeout:     time.Minute,
	}
}

func setup(t *testing.T) (current string, back func()) {
	folder, err := ioutil.TempDir("", "")
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
