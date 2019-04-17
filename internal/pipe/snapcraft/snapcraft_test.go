package snapcraft

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestRunPipeMissingInfo(t *testing.T) {
	for eerr, snap := range map[error]config.Snapcraft{
		ErrNoSummary: {
			Description: "dummy desc",
		},
		ErrNoDescription: {
			Summary: "dummy summary",
		},
		pipe.Skip("no summary nor description were provided"): {},
	} {
		t.Run(fmt.Sprintf("testing if %v happens", eerr), func(t *testing.T) {
			var ctx = &context.Context{
				Config: config.Project{
					Snapcraft: snap,
				},
			}
			assert.Equal(t, eerr, Pipe{}.Run(ctx))
		})
	}
}

func TestRunPipe(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, err)
	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Snapcraft: config.Snapcraft{
			NameTemplate: "foo_{{.Arch}}",
			Summary:      "test summary",
			Description:  "test description",
			Publish:      true,
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "v1.2.3"
	addBinaries(t, ctx, "mybin", dist, "mybin")
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Len(t, ctx.Artifacts.Filter(artifact.ByType(artifact.PublishableSnapcraft)).List(), 2)
}

func TestRunPipeInvalidNameTemplate(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, err)
	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Snapcraft: config.Snapcraft{
			NameTemplate: "foo_{{.Arch}",
			Summary:      "test summary",
			Description:  "test description",
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "v1.2.3"
	addBinaries(t, ctx, "mybin", dist, "mybin")
	assert.EqualError(t, Pipe{}.Run(ctx), `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunPipeWithName(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, err)
	var ctx = context.New(config.Project{
		ProjectName: "testprojectname",
		Dist:        dist,
		Snapcraft: config.Snapcraft{
			NameTemplate: "foo_{{.Arch}}",
			Name:         "testsnapname",
			License:      "MIT",
			Summary:      "test summary",
			Description:  "test description",
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "v1.2.3"
	addBinaries(t, ctx, "testprojectname", dist, "mybin")
	assert.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := ioutil.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	assert.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	assert.NoError(t, err)
	assert.Equal(t, "testsnapname", metadata.Name)
	assert.Equal(t, "MIT", metadata.License)
	assert.Equal(t, "mybin", metadata.Apps["mybin"].Command)
	assert.Equal(t, "mybin", metadata.Apps["testsnapname"].Command)
}

func TestRunPipeWithBinaryInDir(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, err)
	var ctx = context.New(config.Project{
		ProjectName: "testprojectname",
		Dist:        dist,
		Snapcraft: config.Snapcraft{
			NameTemplate: "foo_{{.Arch}}",
			Name:         "testsnapname",
			Summary:      "test summary",
			Description:  "test description",
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "v1.2.3"
	addBinaries(t, ctx, "testprojectname", dist, "bin/mybin")
	assert.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := ioutil.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	assert.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	assert.NoError(t, err)
	assert.Equal(t, "testsnapname", metadata.Name)
	assert.Equal(t, "", metadata.License)
	assert.Equal(t, "mybin", metadata.Apps["mybin"].Command)
	assert.Equal(t, "mybin", metadata.Apps["testsnapname"].Command)
}

func TestRunPipeMetadata(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, err)
	var ctx = context.New(config.Project{
		ProjectName: "testprojectname",
		Dist:        dist,
		Snapcraft: config.Snapcraft{
			NameTemplate: "foo_{{.Arch}}",
			Summary:      "test summary",
			Description:  "test description",
			Apps: map[string]config.SnapcraftAppMetadata{
				"mybin": {
					Plugs:  []string{"home", "network", "personal-files"},
					Daemon: "simple",
					Args:   "--foo --bar",
				},
			},
			Plugs: map[string]interface{}{
				"personal-files": map[string]interface{}{
					"read": []string{"$HOME/test"},
				},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "v1.2.3"
	addBinaries(t, ctx, "mybin", dist, "mybin")
	assert.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := ioutil.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	assert.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	assert.NoError(t, err)
	assert.Equal(t, []string{"home", "network", "personal-files"}, metadata.Apps["mybin"].Plugs)
	assert.Equal(t, "simple", metadata.Apps["mybin"].Daemon)
	assert.Equal(t, "mybin --foo --bar", metadata.Apps["mybin"].Command)
	assert.Equal(t, []string{"home", "network", "personal-files"}, metadata.Apps["testprojectname"].Plugs)
	assert.Equal(t, "simple", metadata.Apps["testprojectname"].Daemon)
	assert.Equal(t, "mybin --foo --bar", metadata.Apps["testprojectname"].Command)
	assert.Equal(t, map[interface{}]interface{}(map[interface{}]interface{}{"read": []interface{}{"$HOME/test"}}), metadata.Plugs["personal-files"])
}

func TestNoSnapcraftInPath(t *testing.T) {
	var path = os.Getenv("PATH")
	defer func() {
		assert.NoError(t, os.Setenv("PATH", path))
	}()
	assert.NoError(t, os.Setenv("PATH", ""))
	var ctx = context.New(config.Project{
		Snapcraft: config.Snapcraft{
			Summary:     "dummy",
			Description: "dummy",
		},
	})
	assert.EqualError(t, Pipe{}.Run(ctx), ErrNoSnapcraft.Error())
}

func TestDefault(t *testing.T) {
	var ctx = context.New(config.Project{})
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, defaultNameTemplate, ctx.Config.Snapcraft.NameTemplate)
}

func TestPublish(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.Artifacts.Add(artifact.Artifact{
		Name:   "mybin",
		Path:   "nope.snap",
		Goarch: "amd64",
		Goos:   "linux",
		Type:   artifact.PublishableSnapcraft,
	})
	err := Pipe{}.Publish(ctx)
	assert.Contains(t, err.Error(), "failed to push nope.snap package")
}

func TestDefaultSet(t *testing.T) {
	var ctx = context.New(config.Project{
		Snapcraft: config.Snapcraft{
			NameTemplate: "foo",
		},
	})
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "foo", ctx.Config.Snapcraft.NameTemplate)
}

func addBinaries(t *testing.T, ctx *context.Context, name, dist, dest string) {
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "386", "arm6"} {
			var folder = goos + goarch
			assert.NoError(t, os.Mkdir(filepath.Join(dist, folder), 0755))
			var binPath = filepath.Join(dist, folder, name)
			_, err := os.Create(binPath)
			assert.NoError(t, err)
			ctx.Artifacts.Add(artifact.Artifact{
				Name:   dest,
				Path:   binPath,
				Goarch: goarch,
				Goos:   goos,
				Type:   artifact.Binary,
			})
		}
	}
}
