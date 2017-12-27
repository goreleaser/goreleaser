package fpm

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestRunPipeNoFormats(t *testing.T) {
	var ctx = &context.Context{
		Version:     "1.0.0",
		Config:      config.Project{},
		Parallelism: runtime.NumCPU(),
	}
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestRunPipe(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0755))
	var binPath = filepath.Join(dist, "mybin", "mybin")
	_, err = os.Create(binPath)
	assert.NoError(t, err)
	var ctx = &context.Context{
		Version:     "1.0.0",
		Parallelism: runtime.NumCPU(),
		Debug:       true,
		Artifacts:   artifact.New(),
		Config: config.Project{
			ProjectName: "mybin",
			Dist:        dist,
			FPM: config.FPM{
				NameTemplate: defaultNameTemplate,
				Formats:      []string{"deb", "rpm"},
				Dependencies: []string{"make"},
				Conflicts:    []string{"git"},
				Description:  "Some description",
				License:      "MIT",
				Maintainer:   "me@me",
				Vendor:       "asdf",
				Homepage:     "https://goreleaser.github.io",
			},
		},
	}
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "386"} {
			ctx.Artifacts.Add(artifact.Artifact{
				Name:   "mybin",
				Path:   binPath,
				Goarch: goarch,
				Goos:   goos,
				Type:   artifact.Binary,
			})
		}
	}
	assert.NoError(t, Pipe{}.Run(ctx))
}

func TestNoFPMInPath(t *testing.T) {
	var path = os.Getenv("PATH")
	defer func() {
		assert.NoError(t, os.Setenv("PATH", path))
	}()
	assert.NoError(t, os.Setenv("PATH", ""))
	var ctx = &context.Context{
		Version:     "1.0.0",
		Parallelism: runtime.NumCPU(),
		Config: config.Project{
			FPM: config.FPM{
				Formats: []string{"deb", "rpm"},
			},
		},
	}
	assert.EqualError(t, Pipe{}.Run(ctx), ErrNoFPM.Error())
}

func TestInvalidNameTemplate(t *testing.T) {
	var ctx = &context.Context{
		Parallelism: runtime.NumCPU(),
		Artifacts:   artifact.New(),
		Config: config.Project{
			FPM: config.FPM{
				NameTemplate: "{{.Foo}",
				Formats: []string{"deb"},
			},
		},
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Name:   "mybin",
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.Binary,
	})
	assert.Contains(t, Pipe{}.Run(ctx).Error(), `template: {{.Foo}:1: unexpected "}" in operand`)
}


func TestCreateFileDoesntExist(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0755))
	var ctx = &context.Context{
		Version:     "1.0.0",
		Parallelism: runtime.NumCPU(),
		Artifacts:   artifact.New(),
		Config: config.Project{
			Dist: dist,
			FPM: config.FPM{
				Formats: []string{"deb", "rpm"},
				Files: map[string]string{
					"testdata/testfile.txt": "/var/lib/test/testfile.txt",
				},
			},
		},
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Name:   "mybin",
		Path:   filepath.Join(dist, "mybin", "mybin"),
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.Binary,
	})
	assert.Contains(t, Pipe{}.Run(ctx).Error(), `dist/mybin/mybin', does it exist?`)
}

func TestCmd(t *testing.T) {
	cmd := cmd([]string{"--help"})
	assert.NotEmpty(t, cmd.Env)
	assert.Contains(t, cmd.Env[0], gnuTarPath)
}

func TestDefault(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			FPM: config.FPM{},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "/usr/local/bin", ctx.Config.FPM.Bindir)
	assert.Equal(t, defaultNameTemplate, ctx.Config.FPM.NameTemplate)
}

func TestDefaultSet(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			FPM: config.FPM{
				Bindir: "/bin",
				NameTemplate: "foo",
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "/bin", ctx.Config.FPM.Bindir)
	assert.Equal(t, "foo", ctx.Config.FPM.NameTemplate)
}
