package archive

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

func TestRunPipe(t *testing.T) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(err)
	current, err := os.Getwd()
	assert.NoError(err)
	assert.NoError(os.Chdir(folder))
	defer func() {
		assert.NoError(os.Chdir(current))
	}()
	var dist = filepath.Join(folder, "dist")
	assert.NoError(os.Mkdir(dist, 0755))
	assert.NoError(os.Mkdir(filepath.Join(dist, "mybin"), 0755))
	_, err = os.Create(filepath.Join(dist, "mybin", "mybin"))
	assert.NoError(err)
	readme, err := os.Create(filepath.Join(folder, "README.md"))
	assert.NoError(err)
	var ctx = &context.Context{
		Archives: map[string]string{
			"darwinamd64":  "mybin",
			"windowsamd64": "mybin",
		},
		Config: config.Project{
			Dist: dist,
			Archive: config.Archive{
				Files: []string{
					"README.md",
				},
				FormatOverrides: []config.FormatOverride{
					{
						Goos:   "windows",
						Format: "zip",
					},
				},
			},
		},
	}
	for _, format := range []string{"tar.gz", "zip"} {
		t.Run("Archive format "+format, func(t *testing.T) {
			ctx.Config.Archive.Format = format
			assert.NoError(Pipe{}.Run(ctx))
		})
	}
	t.Run("Removed README", func(t *testing.T) {
		assert.NoError(os.Remove(readme.Name()))
		assert.Error(Pipe{}.Run(ctx))
	})
}

func TestRunPipeDistRemoved(t *testing.T) {
	var assert = assert.New(t)
	var ctx = &context.Context{
		Archives: map[string]string{
			"darwinamd64":  "mybin",
			"windowsamd64": "mybin",
		},
		Config: config.Project{
			Dist: "/path/nope",
			Archive: config.Archive{
				Format: "zip",
			},
		},
	}
	assert.Error(Pipe{}.Run(ctx))
}

func TestFormatFor(t *testing.T) {
	var assert = assert.New(t)
	var ctx = &context.Context{
		Config: config.Project{
			Archive: config.Archive{
				Format: "tar.gz",
				FormatOverrides: []config.FormatOverride{
					{
						Goos:   "windows",
						Format: "zip",
					},
				},
			},
		},
	}
	assert.Equal("zip", formatFor(ctx, "windowsamd64"))
	assert.Equal("tar.gz", formatFor(ctx, "linux386"))
}
