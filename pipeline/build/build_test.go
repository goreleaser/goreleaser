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
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
)

var emptyEnv []string

func TestPipeDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestRun(t *testing.T) {
	assert.NoError(t, run(buildtarget.Runtime, []string{"go", "list", "./..."}, emptyEnv))
}

func TestRunInvalidCommand(t *testing.T) {
	assert.Error(t, run(buildtarget.Runtime, []string{"gggggo", "nope"}, emptyEnv))
}

func TestBuild(t *testing.T) {
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
	assert.NoError(t, doBuild(ctx, ctx.Config.Builds[0], buildtarget.Runtime))
}

func TestRunFullPipe(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)
	var binary = filepath.Join(folder, buildtarget.Runtime.String(), "testing")
	var pre = filepath.Join(folder, "pre")
	var post = filepath.Join(folder, "post")
	var config = config.Project{
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
		Archive: config.Archive{
			Replacements: map[string]string{
				"linux":  "linuxx",
				"darwin": "darwinn",
			},
		},
	}
	var ctx = context.New(config)
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Len(t, ctx.Artifacts.List(), 1)
	assert.True(t, exists(binary), binary)
	assert.True(t, exists(pre), pre)
	assert.True(t, exists(post), post)
}

func TestRunPipeArmBuilds(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)
	var binary = filepath.Join(folder, "linuxarm6", "armtesting")
	var config = config.Project{
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
	var ctx = context.New(config)
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Len(t, ctx.Artifacts.List(), 2)
	assert.True(t, exists(binary), binary)
}

func TestBuildFailed(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)
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
	var ctx = context.New(config)
	assertContainsError(t, Pipe{}.Run(ctx), `flag provided but not defined: -flag-that-dont-exists-to-force-failure`)
	assert.Empty(t, ctx.Artifacts.List())
}

func TestRunPipeWithInvalidOS(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)
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
	assert.NoError(t, Pipe{}.Run(context.New(config)))
}

func TestRunInvalidLdflags(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)
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
	assert.EqualError(t, Pipe{}.Run(context.New(config)), `template: ldflags:1: unexpected "}" in operand`)
}

func TestRunPipeFailingHooks(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)
	var config = config.Project{
		Builds: []config.Build{
			{
				Binary: "hooks",
				Hooks:  config.Hooks{},
				Goos: []string{
					runtime.GOOS,
				},
				Goarch: []string{
					runtime.GOARCH,
				},
			},
		},
	}
	t.Run("pre-hook", func(t *testing.T) {
		var ctx = context.New(config)
		ctx.Config.Builds[0].Hooks.Pre = "exit 1"
		ctx.Config.Builds[0].Hooks.Post = "echo post"
		assert.EqualError(t, Pipe{}.Run(ctx), `pre hook failed: `)
	})
	t.Run("post-hook", func(t *testing.T) {
		var ctx = context.New(config)
		ctx.Config.Builds[0].Hooks.Pre = "echo pre"
		ctx.Config.Builds[0].Hooks.Post = "exit 1"
		assert.EqualError(t, Pipe{}.Run(ctx), `post hook failed: `)
	})
}

func TestRunPipeWithouMainFunc(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeMainWithoutMainFunc(t, folder)
	var config = config.Project{
		Builds: []config.Build{
			{
				Binary: "no-main",
				Hooks:  config.Hooks{},
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
	t.Run("empty", func(t *testing.T) {
		ctx.Config.Builds[0].Main = ""
		assert.EqualError(t, Pipe{}.Run(ctx), `build for no-main does not contain a main function`)
	})
	t.Run("not main.go", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "foo.go"
		assert.EqualError(t, Pipe{}.Run(ctx), `could not open foo.go: stat foo.go: no such file or directory`)
	})
	t.Run("glob", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "."
		assert.EqualError(t, Pipe{}.Run(ctx), `build for no-main does not contain a main function`)
	})
	t.Run("fixed main.go", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "main.go"
		assert.EqualError(t, Pipe{}.Run(ctx), `build for no-main does not contain a main function`)
	})
}

func TestRunPipeWithMainFuncNotInMainGoFile(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	assert.NoError(t, ioutil.WriteFile(
		filepath.Join(folder, "foo.go"),
		[]byte("package main\nfunc main() {println(0)}"),
		0644,
	))
	var config = config.Project{
		Builds: []config.Build{
			{
				Binary: "foo",
				Hooks:  config.Hooks{},
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
	t.Run("empty", func(t *testing.T) {
		ctx.Config.Builds[0].Main = ""
		assert.NoError(t, Pipe{}.Run(ctx))
	})
	t.Run("foo.go", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "foo.go"
		assert.NoError(t, Pipe{}.Run(ctx))
	})
	t.Run("glob", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "."
		assert.NoError(t, Pipe{}.Run(ctx))
	})
}

func TestDefaultNoBuilds(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
}

func TestDefaultEmptyBuild(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Release: config.Release{
				GitHub: config.Repo{
					Name: "foo",
				},
			},
			Builds: []config.Build{
				{},
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	var build = ctx.Config.Builds[0]
	assert.Equal(t, ctx.Config.Release.GitHub.Name, build.Binary)
	assert.Equal(t, ".", build.Main)
	assert.Equal(t, []string{"linux", "darwin"}, build.Goos)
	assert.Equal(t, []string{"amd64", "386"}, build.Goarch)
	assert.Equal(t, []string{"6"}, build.Goarm)
	assert.Equal(t, "-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}", build.Ldflags)
}

func TestDefaultPartialBuilds(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Builds: []config.Build{
				{
					Binary: "bar",
					Goos:   []string{"linux"},
					Main:   "./cmd/main.go",
				},
				{
					Binary:  "foo",
					Ldflags: "-s -w",
					Goarch:  []string{"386"},
				},
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	t.Run("build0", func(t *testing.T) {
		var build = ctx.Config.Builds[0]
		assert.Equal(t, "bar", build.Binary)
		assert.Equal(t, "./cmd/main.go", build.Main)
		assert.Equal(t, []string{"linux"}, build.Goos)
		assert.Equal(t, []string{"amd64", "386"}, build.Goarch)
		assert.Equal(t, []string{"6"}, build.Goarm)
		assert.Equal(t, "-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}", build.Ldflags)
	})
	t.Run("build1", func(t *testing.T) {
		var build = ctx.Config.Builds[1]
		assert.Equal(t, "foo", build.Binary)
		assert.Equal(t, ".", build.Main)
		assert.Equal(t, []string{"linux", "darwin"}, build.Goos)
		assert.Equal(t, []string{"386"}, build.Goarch)
		assert.Equal(t, []string{"6"}, build.Goarm)
		assert.Equal(t, "-s -w", build.Ldflags)
	})
}

func TestDefaultFillSingleBuild(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()

	var ctx = &context.Context{
		Config: config.Project{
			Release: config.Release{
				GitHub: config.Repo{
					Name: "foo",
				},
			},
			SingleBuild: config.Build{
				Main: "testreleaser",
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Len(t, ctx.Config.Builds, 1)
	assert.Equal(t, ctx.Config.Builds[0].Binary, "foo")
}

func exists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}

func writeMainWithoutMainFunc(t *testing.T, folder string) {
	assert.NoError(t, ioutil.WriteFile(
		filepath.Join(folder, "main.go"),
		[]byte("package main\nconst a = 2\nfunc notMain() {println(0)}"),
		0644,
	))
}

func writeGoodMain(t *testing.T, folder string) {
	assert.NoError(t, ioutil.WriteFile(
		filepath.Join(folder, "main.go"),
		[]byte("package main\nvar a = 1\nfunc main() {println(0)}"),
		0644,
	))
}

func assertContainsError(t *testing.T, err error, s string) {
	assert.Error(t, err)
	assert.Contains(t, err.Error(), s)
}
