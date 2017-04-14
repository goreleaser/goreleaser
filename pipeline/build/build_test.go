package build

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestPipeDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestRun(t *testing.T) {
	assert.NoError(t, run(runtime.GOOS, runtime.GOARCH, []string{"go", "list", "./..."}))
}

func TestRunInvalidCommand(t *testing.T) {
	assert.Error(t, run(runtime.GOOS, runtime.GOARCH, []string{"gggggo", "nope"}))
}

func TestBuild(t *testing.T) {
	assert := assert.New(t)
	var config = config.Project{
		Build: config.Build{
			Binary: "testing",
			Flags:  "-n",
		},
	}
	var ctx = &context.Context{
		Config: config,
	}
	assert.NoError(build("build_test", runtime.GOOS, runtime.GOARCH, ctx))
}

func TestRunFullPipe(t *testing.T) {
	assert := assert.New(t)
	folder, err := ioutil.TempDir("", "gorelasertest")
	assert.NoError(err)
	var binary = filepath.Join(folder, "testing")
	var pre = filepath.Join(folder, "pre")
	var post = filepath.Join(folder, "post")
	var config = config.Project{
		Dist: folder,
		Build: config.Build{
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
	}
	var ctx = &context.Context{
		Config:   config,
		Archives: map[string]string{},
	}
	assert.NoError(Pipe{}.Run(ctx))
	assert.True(exists(binary), binary)
	assert.True(exists(pre), pre)
	assert.True(exists(post), post)
}

func TestRunPipeWithInvalidOS(t *testing.T) {
	assert := assert.New(t)
	var config = config.Project{
		Build: config.Build{
			Flags: "-v",
			Goos: []string{
				"windows",
			},
			Goarch: []string{
				"arm",
			},
		},
	}
	var ctx = &context.Context{
		Config:   config,
		Archives: map[string]string{},
	}
	assert.NoError(Pipe{}.Run(ctx))
}

func TestRunPipeFailingHooks(t *testing.T) {
	assert := assert.New(t)
	var config = config.Project{
		Build: config.Build{
			Hooks: config.Hooks{},
			Goos: []string{
				runtime.GOOS,
			},
			Goarch: []string{
				runtime.GOARCH,
			},
		},
	}
	var ctx = &context.Context{
		Config:   config,
		Archives: map[string]string{},
	}
	t.Run("pre-hook", func(t *testing.T) {
		ctx.Config.Build.Hooks.Pre = "exit 1"
		assert.Error(Pipe{}.Run(ctx))
	})
	t.Run("post-hook", func(t *testing.T) {
		ctx.Config.Build.Hooks.Post = "exit 1"
		assert.Error(Pipe{}.Run(ctx))
	})
}

func exists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}
