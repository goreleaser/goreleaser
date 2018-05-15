package golang

import (
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	api "github.com/goreleaser/goreleaser/build"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
)

var runtimeTarget = runtime.GOOS + "_" + runtime.GOARCH

func TestWithDefaults(t *testing.T) {
	for name, testcase := range map[string]struct {
		build   config.Build
		targets []string
	}{
		"full": {
			build: config.Build{
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
			targets: []string{
				"linux_amd64",
				"darwin_amd64",
				"windows_amd64",
				"linux_arm_6",
			},
		},
		"empty": {
			build: config.Build{
				Binary: "foo",
			},
			targets: []string{
				"linux_amd64",
				"linux_386",
				"darwin_amd64",
				"darwin_386",
			},
		},
	} {
		t.Run(name, func(tt *testing.T) {
			var config = config.Project{
				Builds: []config.Build{
					testcase.build,
				},
			}
			var ctx = context.New(config)
			var build = Default.WithDefaults(ctx.Config.Builds[0])
			assert.ElementsMatch(t, build.Targets, testcase.targets)
		})
	}
}

func TestBuild(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)
	var config = config.Project{
		Builds: []config.Build{
			{
				Binary: "foo",
				Targets: []string{
					"linux_amd64",
					"darwin_amd64",
					"windows_amd64",
					"linux_arm_6",
				},
				Asmflags: []string{".=", "all="},
				Gcflags:  []string{"all="},
			},
		},
	}
	var ctx = context.New(config)
	var build = ctx.Config.Builds[0]
	for _, target := range build.Targets {
		var ext string
		if strings.HasPrefix(target, "windows") {
			ext = ".exe"
		}
		var err = Default.Build(ctx, build, api.Options{
			Target: target,
			Name:   build.Binary,
			Path:   filepath.Join(folder, "dist", target, build.Binary),
			Ext:    ext,
		})
		assert.NoError(t, err)
	}
	assert.ElementsMatch(t, ctx.Artifacts.List(), []artifact.Artifact{
		{
			Name:   "foo",
			Path:   filepath.Join(folder, "dist", "linux_amd64", "foo"),
			Goos:   "linux",
			Goarch: "amd64",
			Type:   artifact.Binary,
			Extra: map[string]string{
				"Ext":    "",
				"Binary": "foo",
			},
		},
		{
			Name:   "foo",
			Path:   filepath.Join(folder, "dist", "darwin_amd64", "foo"),
			Goos:   "darwin",
			Goarch: "amd64",
			Type:   artifact.Binary,
			Extra: map[string]string{
				"Ext":    "",
				"Binary": "foo",
			},
		},
		{
			Name:   "foo",
			Path:   filepath.Join(folder, "dist", "linux_arm_6", "foo"),
			Goos:   "linux",
			Goarch: "arm",
			Goarm:  "6",
			Type:   artifact.Binary,
			Extra: map[string]string{
				"Ext":    "",
				"Binary": "foo",
			},
		},
		{
			Name:   "foo",
			Path:   filepath.Join(folder, "dist", "windows_amd64", "foo"),
			Goos:   "windows",
			Goarch: "amd64",
			Type:   artifact.Binary,
			Extra: map[string]string{
				"Ext":    ".exe",
				"Binary": "foo",
			},
		},
	})
}

func TestBuildFailed(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)
	var config = config.Project{
		Builds: []config.Build{
			{
				Flags: []string{"-flag-that-dont-exists-to-force-failure"},
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

func TestBuildInvalidTarget(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)
	var target = "linux"
	var config = config.Project{
		Builds: []config.Build{
			{
				Binary:  "foo",
				Targets: []string{target},
			},
		},
	}
	var ctx = context.New(config)
	var build = ctx.Config.Builds[0]
	var err = Default.Build(ctx, build, api.Options{
		Target: target,
		Name:   build.Binary,
		Path:   filepath.Join(folder, "dist", target, build.Binary),
	})
	assert.EqualError(t, err, "linux is not a valid build target")
	assert.Len(t, ctx.Artifacts.List(), 0)
}

func TestRunInvalidAsmflags(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)
	var config = config.Project{
		Builds: []config.Build{
			{
				Binary:   "nametest",
				Asmflags: []string{"{{.Version}"},
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
	assert.EqualError(t, err, `template: asmflags:1: unexpected "}" in operand`)
}

func TestRunInvalidGcflags(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)
	var config = config.Project{
		Builds: []config.Build{
			{
				Binary:  "nametest",
				Gcflags: []string{"{{.Version}"},
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
	assert.EqualError(t, err, `template: gcflags:1: unexpected "}" in operand`)
}

func TestRunInvalidLdflags(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)
	var config = config.Project{
		Builds: []config.Build{
			{
				Binary:  "nametest",
				Flags:   []string{"-v"},
				Ldflags: []string{"-s -w -X main.version={{.Version}"},
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
		}), `stat foo.go: no such file or directory`)
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

func TestLdFlagsFullTemplate(t *testing.T) {
	var config = config.Project{
		Builds: []config.Build{
			{
				Ldflags: []string{
					`-s -w -X main.version={{.Version}} -X main.tag={{.Tag}} -X main.date={{.Date}} -X main.commit={{.Commit}} -X "main.foo={{.Env.FOO}}" -X main.time={{ time "20060102" }}`,
				},
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
	flags, err := processField(ctx, ctx.Config.Builds[0].Ldflags[0], "ldflags")
	assert.NoError(t, err)
	assert.Contains(t, flags, "-s -w")
	assert.Contains(t, flags, "-X main.version=1.2.3")
	assert.Contains(t, flags, "-X main.tag=v1.2.3")
	assert.Contains(t, flags, "-X main.commit=123")
	// TODO: this will break in 2019
	assert.Contains(t, flags, "-X main.date=2018")
	assert.Contains(t, flags, "-X main.time=2018")
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
					{Ldflags: []string{template}},
				},
			}
			var ctx = &context.Context{
				Config: config,
			}
			flags, err := processField(ctx, template, "ldflags")
			assert.EqualError(tt, err, eerr)
			assert.Empty(tt, flags)
		})
	}
}

func TestProcessFlags(t *testing.T) {
	var ctx = &context.Context{
		Version: "1.2.3",
	}

	var source = []string{
		"{{.Version}}",
		"flag",
	}

	var expected = []string{
		"-testflag=1.2.3",
		"-testflag=flag",
	}

	flags, err := processFlags(ctx, source, "testflag", "-testflag=")
	assert.NoError(t, err)
	assert.Len(t, flags, 2)
	assert.Equal(t, expected, flags)
}

func TestProcessFlagsInvalid(t *testing.T) {
	var ctx = &context.Context{}

	var source = []string{
		"{{.Version}",
	}

	var expected = `template: testflag:1: unexpected "}" in operand`

	flags, err := processFlags(ctx, source, "testflag", "-testflag=")
	assert.EqualError(t, err, expected)
	assert.Nil(t, flags)
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
