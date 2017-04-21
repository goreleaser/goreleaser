package fpm

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestRunPipeNoFormats(t *testing.T) {
	var assert = assert.New(t)
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(Pipe{}.Run(ctx))
}

func TestRunPipe(t *testing.T) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(os.Mkdir(dist, 0755))
	assert.NoError(os.Mkdir(filepath.Join(dist, "mybin"), 0755))
	_, err = os.Create(filepath.Join(dist, "mybin", "mybin"))
	assert.NoError(err)
	var ctx = &context.Context{
		Archives: map[string]string{
			"linuxamd64": "mybin",
		},
		Config: config.Project{
			Dist: dist,
			Build: config.Build{
				Goarch: []string{
					"amd64",
					"i386",
				},
				Binary: "mybin",
			},
			FPM: config.FPM{
				Formats:      []string{"deb"},
				Dependencies: []string{"make"},
				Conflicts:    []string{"git"},
				Options: config.FPMOptions{
					Description: "Some description",
					License:     "MIT",
					Maintainer:  "me@me",
					Vendor:      "asdf",
					URL:         "https://goreleaser.github.io",
				},
			},
		},
	}
	assert.NoError(Pipe{}.Run(ctx))
}

func TestNoFPMInPath(t *testing.T) {
	var assert = assert.New(t)
	var path = os.Getenv("PATH")
	defer func() {
		assert.NoError(os.Setenv("PATH", path))
	}()
	assert.NoError(os.Setenv("PATH", ""))
	var ctx = &context.Context{
		Config: config.Project{
			FPM: config.FPM{
				Formats: []string{"deb"},
			},
		},
	}
	assert.EqualError(Pipe{}.Run(ctx), ErrNoFPM.Error())
}
