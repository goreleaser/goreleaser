package build

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/buildtarget"
	"github.com/stretchr/testify/assert"
)

var emptyEnv []string

func TestPipeDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestRun(t *testing.T) {
	assert.NoError(t, run(buildtarget.Runtime, []string{"go", "list", "./..."}, emptyEnv))
}

func TestRunInvalidCommand(t *testing.T) {
	assert.Error(t, run(buildtarget.Runtime, []string{"gggggo", "nope"}, emptyEnv))
}

func TestBuild(t *testing.T) {
	assert := assert.New(t)
	var config = config.Project{
		Builds: []config.Build{
			{
				Binary: "testing",
				Flags:  "-n",
				Env:    []string{"BLAH=1"},
			},
		},
	}
	var ctx = context.New(config)
	assert.NoError(doBuild(ctx, ctx.Config.Builds[0], buildtarget.Runtime))
}

func TestRunFullPipe(t *testing.T) {
	assert := assert.New(t)
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(err)
	var binary = filepath.Join(folder, "testing")
	var pre = filepath.Join(folder, "pre")
	var post = filepath.Join(folder, "post")
	var config = config.Project{
		Dist: folder,
		Builds: []config.Build{
			{
				Binary:  "testing",
				Flags:   "-v",
				Ldflags: "-X main.test=testing",
				Hooks: config.Hooks{
					Pre:  "touch " + pre,
					Post: "touch " + post,
				},
				Goos: []string{
					runtime.GOOS,
				},
				Goarch: []string{
					runtime.GOARCH,
				},
			},
		},
	}
	assert.NoError(Pipe{}.Run(context.New(config)))
	assert.True(exists(binary), binary)
	assert.True(exists(pre), pre)
	assert.True(exists(post), post)
}

func TestRunPipeFormatBinary(t *testing.T) {
	assert := assert.New(t)
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(err)
	var binary = filepath.Join(folder, "binary-testing")
	var config = config.Project{
		ProjectName: "testing",
		Dist:        folder,
		Builds: []config.Build{
			{
				Binary: "testing",
				Goos: []string{
					runtime.GOOS,
				},
				Goarch: []string{
					runtime.GOARCH,
				},
			},
		},
		Archive: config.Archive{
			Format:       "binary",
			NameTemplate: "binary-{{.Binary}}",
		},
	}
	assert.NoError(Pipe{}.Run(context.New(config)))
	assert.True(exists(binary))
}

func TestRunPipeArmBuilds(t *testing.T) {
	assert := assert.New(t)
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(err)
	var binary = filepath.Join(folder, "armtesting")
	var config = config.Project{
		Dist: folder,
		Builds: []config.Build{
			{
				Binary:  "armtesting",
				Flags:   "-v",
				Ldflags: "-X main.test=armtesting",
				Goos: []string{
					"linux",
				},
				Goarch: []string{
					"arm",
					"arm64",
				},
				Goarm: []string{
					"6",
				},
			},
		},
	}
	assert.NoError(Pipe{}.Run(context.New(config)))
	assert.True(exists(binary), binary)
}

func TestBuildFailed(t *testing.T) {
	assert := assert.New(t)
	var config = config.Project{
		Builds: []config.Build{
			{
				Flags: "-flag-that-dont-exists-to-force-failure",
				Goos: []string{
					runtime.GOOS,
				},
				Goarch: []string{
					runtime.GOARCH,
				},
			},
		},
	}
	assert.Error(Pipe{}.Run(context.New(config)))
}

func TestRunPipeWithInvalidOS(t *testing.T) {
	assert := assert.New(t)
	var config = config.Project{
		Builds: []config.Build{
			{
				Flags: "-v",
				Goos: []string{
					"windows",
				},
				Goarch: []string{
					"arm",
				},
			},
		},
	}
	assert.NoError(Pipe{}.Run(context.New(config)))
}

func TestRunInvalidNametemplate(t *testing.T) {
	var assert = assert.New(t)
	for _, format := range []string{"tar.gz", "zip", "binary"} {
		var config = config.Project{
			ProjectName: "nameeeee",
			Builds: []config.Build{
				{
					Binary: "namet{{.est}",
					Flags:  "-v",
					Goos: []string{
						runtime.GOOS,
					},
					Goarch: []string{
						runtime.GOARCH,
					},
				},
			},
			Archive: config.Archive{
				Format:       format,
				NameTemplate: "{{.Binary}",
			},
		}
		assert.Error(Pipe{}.Run(context.New(config)))
	}
}

func TestRunInvalidLdflags(t *testing.T) {
	var assert = assert.New(t)
	var config = config.Project{
		Builds: []config.Build{
			{
				Binary:  "nametest",
				Flags:   "-v",
				Ldflags: "-s -w -X main.version={{.Version}",
				Goos: []string{
					runtime.GOOS,
				},
				Goarch: []string{
					runtime.GOARCH,
				},
			},
		},
	}
	assert.Error(Pipe{}.Run(context.New(config)))
}

func TestRunPipeFailingHooks(t *testing.T) {
	assert := assert.New(t)
	var config = config.Project{
		Builds: []config.Build{
			{
				Hooks: config.Hooks{},
				Goos: []string{
					runtime.GOOS,
				},
				Goarch: []string{
					runtime.GOARCH,
				},
			},
		},
	}
	var ctx = context.New(config)
	t.Run("pre-hook", func(t *testing.T) {
		ctx.Config.Builds[0].Hooks.Pre = "exit 1"
		assert.Error(Pipe{}.Run(ctx))
	})
	t.Run("post-hook", func(t *testing.T) {
		ctx.Config.Builds[0].Hooks.Post = "exit 1"
		assert.Error(Pipe{}.Run(ctx))
	})
}

func exists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}
