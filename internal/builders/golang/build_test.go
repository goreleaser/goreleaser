package golang

import (
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"

	api "github.com/goreleaser/goreleaser/build"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
)

var runtimeTarget = runtime.GOOS + "_" + runtime.GOARCH

func TestBuild(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)
	var config = config.Project{
		Builds: []config.Build{
			{
				Binary: "foo",
				Goos: []string{
					"linux",
					"windows",
					"darwin",
				},
				Goarch: []string{
					"amd64",
					"arm",
				},
				Goarm: []string{
					"6",
				},
			},
		},
	}
	var ctx = context.New(config)
	var build = Default.Default(ctx.Config.Builds[0])
	assert.ElementsMatch(t, build.Targets, []string{
		"linux_amd64",
		"darwin_amd64",
		"windows_amd64",
		"linux_arm_6",
	})
	for _, target := range build.Targets {
		var err = Default.Build(ctx, build, api.Options{
			Target: target,
			Name:   build.Binary,
			Path:   filepath.Join(folder, "dist", target, build.Binary),
		})
		assert.NoError(t, err)
	}

	assert.Len(t, ctx.Artifacts.List(), len(build.Targets))
}

func TestBuildFailed(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)
	var config = config.Project{
		Builds: []config.Build{
			{
				Flags: "-flag-that-dont-exists-to-force-failure",
				Targets: []string{
					runtimeTarget,
				},
			},
		},
	}
	var ctx = context.New(config)
	var err = Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: "darwin_amd64",
	})
	assertContainsError(t, err, `flag provided but not defined: -flag-that-dont-exists-to-force-failure`)
	assert.Empty(t, ctx.Artifacts.List())
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
				Targets: []string{
					runtimeTarget,
				},
			},
		},
	}
	var ctx = context.New(config)
	var err = Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: runtimeTarget,
	})
	assert.EqualError(t, err, `template: ldflags:1: unexpected "}" in operand`)
}

func TestRunPipeWithoutMainFunc(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeMainWithoutMainFunc(t, folder)
	var config = config.Project{
		Builds: []config.Build{
			{
				Binary: "no-main",
				Hooks:  config.Hooks{},
				Targets: []string{
					runtimeTarget,
				},
			},
		},
	}
	var ctx = context.New(config)
	t.Run("empty", func(t *testing.T) {
		ctx.Config.Builds[0].Main = ""
		assert.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}), `build for no-main does not contain a main function`)
	})
	t.Run("not main.go", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "foo.go"
		assert.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}), `could not open foo.go: stat foo.go: no such file or directory`)
	})
	t.Run("glob", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "."
		assert.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}), `build for no-main does not contain a main function`)
	})
	t.Run("fixed main.go", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "main.go"
		assert.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}), `build for no-main does not contain a main function`)
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
				Targets: []string{
					runtimeTarget,
				},
			},
		},
	}
	var ctx = context.New(config)
	t.Run("empty", func(t *testing.T) {
		ctx.Config.Builds[0].Main = ""
		assert.NoError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}))
	})
	t.Run("foo.go", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "foo.go"
		assert.NoError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}))
	})
	t.Run("glob", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "."
		assert.NoError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}))
	})
}

// FIXME: probably should be refactored
func TestRunPipeWithInvalidOS(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)
	var config = config.Project{
		Builds: []config.Build{
			{
				Lang:  "go",
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
	assert.NoError(t, Default.Build(context.New(config), config.Builds[0], api.Options{
		Target: "windows_arm",
	}))
}

func TestLdFlagsFullTemplate(t *testing.T) {
	var config = config.Project{
		Builds: []config.Build{
			{
				Ldflags: `-s -w -X main.version={{.Version}} -X main.tag={{.Tag}} -X main.date={{.Date}} -X main.commit={{.Commit}} -X "main.foo={{.Env.FOO}}"`,
			},
		},
	}
	var ctx = &context.Context{
		Git: context.GitInfo{
			CurrentTag: "v1.2.3",
			Commit:     "123",
		},
		Version: "1.2.3",
		Config:  config,
		Env:     map[string]string{"FOO": "123"},
	}
	flags, err := ldflags(ctx, ctx.Config.Builds[0])
	assert.NoError(t, err)
	assert.Contains(t, flags, "-s -w")
	assert.Contains(t, flags, "-X main.version=1.2.3")
	assert.Contains(t, flags, "-X main.tag=v1.2.3")
	assert.Contains(t, flags, "-X main.commit=123")
	assert.Contains(t, flags, "-X main.date=")
	assert.Contains(t, flags, `-X "main.foo=123"`)
}

func TestInvalidTemplate(t *testing.T) {
	for template, eerr := range map[string]string{
		"{{ .Nope }":    `template: ldflags:1: unexpected "}" in operand`,
		"{{.Env.NOPE}}": `template: ldflags:1:6: executing "ldflags" at <.Env.NOPE>: map has no entry for key "NOPE"`,
	} {
		t.Run(template, func(tt *testing.T) {
			var config = config.Project{
				Builds: []config.Build{
					{Ldflags: template},
				},
			}
			var ctx = &context.Context{
				Config: config,
			}
			flags, err := ldflags(ctx, ctx.Config.Builds[0])
			assert.EqualError(tt, err, eerr)
			assert.Empty(tt, flags)
		})
	}
}

//
// Helpers
//

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
