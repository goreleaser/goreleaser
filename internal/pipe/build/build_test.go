package build

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	api "github.com/goreleaser/goreleaser/pkg/build"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBuilder struct {
	fail bool
}

func (*fakeBuilder) WithDefaults(build config.Build) config.Build {
	return build
}

var errFailedBuild = errors.New("fake builder failed")

func (f *fakeBuilder) Build(ctx *context.Context, build config.Build, options api.Options) error {
	if f.fail {
		return errFailedBuild
	}
	if err := os.MkdirAll(filepath.Dir(options.Path), 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(options.Path, []byte("foo"), 0755); err != nil {
		return err
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: options.Name,
	})
	return nil
}

func init() {
	api.Register("fake", &fakeBuilder{})
	api.Register("fakeFail", &fakeBuilder{
		fail: true,
	})
}

func TestPipeDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestBuild(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	var config = config.Project{
		Dist: folder,
		Builds: []config.Build{
			{
				Lang:   "fake",
				Binary: "testing.v{{.Version}}",
				Flags:  []string{"-n"},
				Env:    []string{"BLAH=1"},
			},
		},
	}
	var ctx = &context.Context{
		Artifacts: artifact.New(),
		Git: context.GitInfo{
			CurrentTag: "v1.2.3",
			Commit:     "123",
		},
		Version: "1.2.3",
		Config:  config,
	}
	opts, err := buildOptionsForTarget(ctx, ctx.Config.Builds[0], "darwin_amd64")
	assert.NoError(t, err)
	error := doBuild(ctx, ctx.Config.Builds[0], *opts)
	assert.NoError(t, error)
}

func TestRunPipe(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	var config = config.Project{
		Dist: folder,
		Builds: []config.Build{
			{
				Lang:    "fake",
				Binary:  "testing",
				Flags:   []string{"-v"},
				Ldflags: []string{"-X main.test=testing"},
				Targets: []string{"whatever"},
			},
		},
	}
	var ctx = context.New(config)
	ctx.Git.CurrentTag = "2.4.5"
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Equal(t, ctx.Artifacts.List(), []*artifact.Artifact{{
		Name: "testing",
	}})
}

func TestRunFullPipe(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	var pre = filepath.Join(folder, "pre")
	var post = filepath.Join(folder, "post")
	var config = config.Project{
		Builds: []config.Build{
			{
				ID:      "build1",
				Lang:    "fake",
				Binary:  "testing",
				Flags:   []string{"-v"},
				Ldflags: []string{"-X main.test=testing"},
				Hooks: config.HookConfig{
					Pre: []config.BuildHook{
						{Cmd: "touch " + pre},
					},
					Post: []config.BuildHook{
						{Cmd: "touch " + post},
					},
				},
				Targets: []string{"whatever"},
			},
		},
		Dist: folder,
	}
	var ctx = context.New(config)
	ctx.Git.CurrentTag = "2.4.5"
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Equal(t, ctx.Artifacts.List(), []*artifact.Artifact{{
		Name: "testing",
	}})
	assert.FileExists(t, post)
	assert.FileExists(t, pre)
	assert.FileExists(t, filepath.Join(folder, "build1_whatever", "testing"))
}

func TestRunFullPipeFail(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	var pre = filepath.Join(folder, "pre")
	var post = filepath.Join(folder, "post")
	var config = config.Project{
		Dist: folder,
		Builds: []config.Build{
			{
				Lang:    "fakeFail",
				Binary:  "testing",
				Flags:   []string{"-v"},
				Ldflags: []string{"-X main.test=testing"},
				Hooks: config.HookConfig{
					Pre: []config.BuildHook{
						{Cmd: "touch " + pre},
					},
					Post: []config.BuildHook{
						{Cmd: "touch " + post},
					},
				},
				Targets: []string{"whatever"},
			},
		},
	}
	var ctx = context.New(config)
	ctx.Git.CurrentTag = "2.4.5"
	assert.EqualError(t, Pipe{}.Run(ctx), errFailedBuild.Error())
	assert.Empty(t, ctx.Artifacts.List())
	assert.FileExists(t, pre)
}

func TestRunPipeFailingHooks(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	var cfg = config.Project{
		Dist: folder,
		Builds: []config.Build{
			{
				Lang:    "fake",
				Binary:  "hooks",
				Hooks:   config.HookConfig{},
				Targets: []string{"whatever"},
			},
		},
	}
	t.Run("pre-hook", func(t *testing.T) {
		var ctx = context.New(cfg)
		ctx.Git.CurrentTag = "2.3.4"
		ctx.Config.Builds[0].Hooks.Pre = []config.BuildHook{{Cmd: "exit 1"}}
		ctx.Config.Builds[0].Hooks.Post = []config.BuildHook{{Cmd: "echo post"}}
		assert.EqualError(t, Pipe{}.Run(ctx), `pre hook failed: "": exec: "exit": executable file not found in $PATH`)
	})
	t.Run("post-hook", func(t *testing.T) {
		var ctx = context.New(cfg)
		ctx.Git.CurrentTag = "2.3.4"
		ctx.Config.Builds[0].Hooks.Pre = []config.BuildHook{{Cmd: "echo pre"}}
		ctx.Config.Builds[0].Hooks.Post = []config.BuildHook{{Cmd: "exit 1"}}
		assert.EqualError(t, Pipe{}.Run(ctx), `post hook failed: "": exec: "exit": executable file not found in $PATH`)
	})
}

func TestDefaultNoBuilds(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
}

func TestDefaultExpandEnv(t *testing.T) {
	assert.NoError(t, os.Setenv("XBAR", "FOOBAR"))
	var ctx = &context.Context{
		Config: config.Project{
			Builds: []config.Build{
				{
					Env: []string{
						"XFOO=bar_$XBAR",
					},
				},
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	var env = ctx.Config.Builds[0].Env[0]
	assert.Equal(t, "XFOO=bar_FOOBAR", env)
}

func TestDefaultEmptyBuild(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			ProjectName: "foo",
			Builds: []config.Build{
				{},
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	var build = ctx.Config.Builds[0]
	assert.Equal(t, ctx.Config.ProjectName, build.ID)
	assert.Equal(t, ctx.Config.ProjectName, build.Binary)
	assert.Equal(t, ".", build.Dir)
	assert.Equal(t, ".", build.Main)
	assert.Equal(t, []string{"linux", "darwin"}, build.Goos)
	assert.Equal(t, []string{"amd64", "386"}, build.Goarch)
	assert.Equal(t, []string{"6"}, build.Goarm)
	assert.Len(t, build.Ldflags, 1)
	assert.Equal(t, "-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}} -X main.builtBy=goreleaser", build.Ldflags[0])
}

func TestDefaultBuildID(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			ProjectName: "foo",
			Builds: []config.Build{
				{
					Binary: "{{.Env.FOO}}",
				},
				{
					Binary: "bar",
				},
			},
		},
	}
	assert.EqualError(t, Pipe{}.Default(ctx), "found 2 builds with the ID 'foo', please fix your config")
	var build = ctx.Config.Builds[0]
	assert.Equal(t, ctx.Config.ProjectName, build.ID)
}

func TestSeveralBuildsWithTheSameID(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Builds: []config.Build{
				{
					ID:     "a",
					Binary: "bar",
				},
				{
					ID:     "a",
					Binary: "foo",
				},
			},
		},
	}
	assert.EqualError(t, Pipe{}.Default(ctx), "found 2 builds with the ID 'a', please fix your config")
}

func TestDefaultPartialBuilds(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Builds: []config.Build{
				{
					ID:     "build1",
					Binary: "bar",
					Goos:   []string{"linux"},
					Main:   "./cmd/main.go",
				},
				{
					ID:      "build2",
					Binary:  "foo",
					Dir:     "baz",
					Ldflags: []string{"-s -w"},
					Goarch:  []string{"386"},
				},
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	t.Run("build0", func(t *testing.T) {
		var build = ctx.Config.Builds[0]
		assert.Equal(t, "bar", build.Binary)
		assert.Equal(t, ".", build.Dir)
		assert.Equal(t, "./cmd/main.go", build.Main)
		assert.Equal(t, []string{"linux"}, build.Goos)
		assert.Equal(t, []string{"amd64", "386"}, build.Goarch)
		assert.Equal(t, []string{"6"}, build.Goarm)
		assert.Len(t, build.Ldflags, 1)
		assert.Equal(t, "-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}} -X main.builtBy=goreleaser", build.Ldflags[0])
	})
	t.Run("build1", func(t *testing.T) {
		var build = ctx.Config.Builds[1]
		assert.Equal(t, "foo", build.Binary)
		assert.Equal(t, ".", build.Main)
		assert.Equal(t, "baz", build.Dir)
		assert.Equal(t, []string{"linux", "darwin"}, build.Goos)
		assert.Equal(t, []string{"386"}, build.Goarch)
		assert.Equal(t, []string{"6"}, build.Goarm)
		assert.Len(t, build.Ldflags, 1)
		assert.Equal(t, "-s -w", build.Ldflags[0])
	})
}

func TestDefaultFillSingleBuild(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()

	var ctx = &context.Context{
		Config: config.Project{
			ProjectName: "foo",
			SingleBuild: config.Build{
				Main: "testreleaser",
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Len(t, ctx.Config.Builds, 1)
	assert.Equal(t, ctx.Config.Builds[0].Binary, "foo")
}

func TestSkipBuild(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	var config = config.Project{
		Dist: folder,
		Builds: []config.Build{
			{
				Skip: true,
			},
		},
	}
	var ctx = context.New(config)
	ctx.Git.CurrentTag = "2.4.5"
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Len(t, ctx.Artifacts.List(), 0)
}

func TestExtWindows(t *testing.T) {
	assert.Equal(t, ".exe", extFor("windows_amd64", config.FlagArray{}))
	assert.Equal(t, ".exe", extFor("windows_386", config.FlagArray{}))
	assert.Equal(t, ".exe", extFor("windows_amd64", config.FlagArray{"-tags=dev", "-v"}))
	assert.Equal(t, ".dll", extFor("windows_amd64", config.FlagArray{"-tags=dev", "-v", "-buildmode=c-shared"}))
	assert.Equal(t, ".dll", extFor("windows_386", config.FlagArray{"-buildmode=c-shared"}))
	assert.Equal(t, ".lib", extFor("windows_amd64", config.FlagArray{"-buildmode=c-archive"}))
	assert.Equal(t, ".lib", extFor("windows_386", config.FlagArray{"-tags=dev", "-v", "-buildmode=c-archive"}))
}

func TestExtWasm(t *testing.T) {
	assert.Equal(t, ".wasm", extFor("js_wasm", config.FlagArray{}))
}

func TestExtOthers(t *testing.T) {
	assert.Empty(t, "", extFor("linux_amd64", config.FlagArray{}))
	assert.Empty(t, "", extFor("linuxwin_386", config.FlagArray{}))
	assert.Empty(t, "", extFor("winasdasd_sad", config.FlagArray{}))
}

func TestTemplate(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.Git = context.GitInfo{
		CurrentTag: "v1.2.3",
		Commit:     "123",
	}
	ctx.Version = "1.2.3"
	ctx.Env = map[string]string{"FOO": "123"}
	binary, err := tmpl.New(ctx).
		Apply(`-s -w -X main.version={{.Version}} -X main.tag={{.Tag}} -X main.date={{.Date}} -X main.commit={{.Commit}} -X "main.foo={{.Env.FOO}}"`)
	assert.NoError(t, err)
	assert.Contains(t, binary, "-s -w")
	assert.Contains(t, binary, "-X main.version=1.2.3")
	assert.Contains(t, binary, "-X main.tag=v1.2.3")
	assert.Contains(t, binary, "-X main.commit=123")
	assert.Contains(t, binary, "-X main.date=")
	assert.Contains(t, binary, `-X "main.foo=123"`)
}

func TestRunHookEnvs(t *testing.T) {
	tmp, back := testlib.Mktmp(t)
	defer back()

	var build = config.Build{
		Env: []string{
			fmt.Sprintf("FOO=%s/foo", tmp),
			fmt.Sprintf("BAR=%s/bar", tmp),
		},
	}

	var opts = api.Options{
		Name:   "binary-name",
		Path:   "./binary-name",
		Target: "darwin_amd64",
	}

	simpleHook := func(cmd string) config.BuildHooks {
		return []config.BuildHook{{Cmd: cmd}}
	}

	t.Run("valid cmd template with ctx env", func(t *testing.T) {
		var err = runHook(context.New(config.Project{
			Builds: []config.Build{
				build,
			},
			Env: []string{
				fmt.Sprintf("CTXFOO=%s/foo", tmp),
			},
		}), opts, []string{}, simpleHook("touch {{ .Env.CTXFOO }}"))
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(tmp, "foo"))
	})

	t.Run("valid cmd template with build env", func(t *testing.T) {
		var err = runHook(context.New(config.Project{
			Builds: []config.Build{
				build,
			},
		}), opts, build.Env, simpleHook("touch {{ .Env.FOO }}"))
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(tmp, "foo"))
	})

	t.Run("valid cmd template with hook env", func(t *testing.T) {
		var err = runHook(context.New(config.Project{
			Builds: []config.Build{
				build,
			},
		}), opts, []string{}, []config.BuildHook{{
			Cmd: "touch {{ .Env.HOOK_ONLY_FOO }}",
			Env: []string{
				fmt.Sprintf("HOOK_ONLY_FOO=%s/hook_only", tmp),
			},
		}})
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(tmp, "hook_only"))
	})

	t.Run("valid cmd template with ctx and build env", func(t *testing.T) {
		var err = runHook(context.New(config.Project{
			Builds: []config.Build{
				build,
			},
			Env: []string{
				fmt.Sprintf("OVER_FOO=%s/ctx_over_build", tmp),
			},
		}), opts, []string{
			fmt.Sprintf("OVER_FOO=%s/build_over_ctx", tmp),
		}, simpleHook("touch {{ .Env.OVER_FOO }}"))
		assert.NoError(t, err)

		assert.FileExists(t, filepath.Join(tmp, "build_over_ctx"))
		assert.NoFileExists(t, filepath.Join(tmp, "ctx_over_build"))
	})

	t.Run("valid cmd template with ctx and hook env", func(t *testing.T) {
		var err = runHook(context.New(config.Project{
			Builds: []config.Build{
				build,
			},
			Env: []string{
				fmt.Sprintf("CTX_OR_HOOK_FOO=%s/ctx_over_hook", tmp),
			},
		}), opts, []string{}, []config.BuildHook{{
			Cmd: "touch {{ .Env.CTX_OR_HOOK_FOO }}",
			Env: []string{
				fmt.Sprintf("CTX_OR_HOOK_FOO=%s/hook_over_ctx", tmp),
			},
		}})
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(tmp, "hook_over_ctx"))
		assert.NoFileExists(t, filepath.Join(tmp, "ctx_over_hook"))
	})

	t.Run("valid cmd template with build and hook env", func(t *testing.T) {
		var err = runHook(context.New(config.Project{
			Builds: []config.Build{
				build,
			},
		}), opts, []string{
			fmt.Sprintf("BUILD_OR_HOOK_FOO=%s/build_over_hook", tmp),
		}, []config.BuildHook{{
			Cmd: "touch {{ .Env.BUILD_OR_HOOK_FOO }}",
			Env: []string{
				fmt.Sprintf("BUILD_OR_HOOK_FOO=%s/hook_over_build", tmp),
			},
		}})
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(tmp, "hook_over_build"))
		assert.NoFileExists(t, filepath.Join(tmp, "build_over_hook"))
	})

	t.Run("valid cmd template with ctx, build and hook env", func(t *testing.T) {
		var err = runHook(context.New(config.Project{
			Builds: []config.Build{
				build,
			},
			Env: []string{
				fmt.Sprintf("CTX_OR_BUILD_OR_HOOK_FOO=%s/ctx_wins", tmp),
			},
		}), opts, []string{
			fmt.Sprintf("CTX_OR_BUILD_OR_HOOK_FOO=%s/build_wins", tmp),
		}, []config.BuildHook{{
			Cmd: "touch {{ .Env.CTX_OR_BUILD_OR_HOOK_FOO }}",
			Env: []string{
				fmt.Sprintf("CTX_OR_BUILD_OR_HOOK_FOO=%s/hook_wins", tmp),
			},
		}})
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(tmp, "hook_wins"))
		assert.NoFileExists(t, filepath.Join(tmp, "ctx_wins"))
		assert.NoFileExists(t, filepath.Join(tmp, "build_wins"))
	})

	t.Run("invalid cmd template", func(t *testing.T) {
		var err = runHook(context.New(config.Project{
			Builds: []config.Build{
				build,
			},
		}), opts, build.Env, simpleHook("touch {{ .Env.FOOss }}"))
		assert.EqualError(t, err, `template: tmpl:1:13: executing "tmpl" at <.Env.FOOss>: map has no entry for key "FOOss"`)
	})

	t.Run("invalid dir template", func(t *testing.T) {
		var err = runHook(context.New(config.Project{
			Builds: []config.Build{
				build,
			},
		}), opts, build.Env, []config.BuildHook{{
			Cmd: "echo something",
			Dir: "{{ .Env.INVALID_ENV }}",
		}})
		assert.EqualError(t, err, `template: tmpl:1:7: executing "tmpl" at <.Env.INVALID_ENV>: map has no entry for key "INVALID_ENV"`)
	})

	t.Run("invalid hook env template", func(t *testing.T) {
		var err = runHook(context.New(config.Project{
			Builds: []config.Build{
				build,
			},
		}), opts, build.Env, []config.BuildHook{{
			Cmd: "echo something",
			Env: []string{
				"TEST={{ .Env.MISSING_ENV }}",
			},
		}})
		assert.EqualError(t, err, `template: tmpl:1:12: executing "tmpl" at <.Env.MISSING_ENV>: map has no entry for key "MISSING_ENV"`)
	})

	t.Run("build env inside shell", func(t *testing.T) {
		var shell = `#!/bin/sh -e
touch "$BAR"`
		err := ioutil.WriteFile(filepath.Join(tmp, "test.sh"), []byte(shell), 0750)
		assert.NoError(t, err)
		err = runHook(context.New(config.Project{
			Builds: []config.Build{
				build,
			},
		}), opts, build.Env, simpleHook("sh test.sh"))
		assert.NoError(t, err)
		assert.FileExists(t, filepath.Join(tmp, "bar"))
	})
}

func TestBuild_hooksKnowGoosGoarch(t *testing.T) {
	tmpDir, back := testlib.Mktmp(t)
	defer back()

	build := config.Build{
		Lang:   "fake",
		Goarch: []string{"amd64"},
		Goos:   []string{"linux"},
		Binary: "testing-goos-goarch.v{{.Version}}",
		Targets: []string{
			"linux_amd64",
		},
		Hooks: config.HookConfig{
			Pre: []config.BuildHook{
				{Cmd: "touch pre-hook-{{.Arch}}-{{.Os}}", Dir: tmpDir},
			},
			Post: config.BuildHooks{
				{Cmd: "touch post-hook-{{.Arch}}-{{.Os}}", Dir: tmpDir},
			},
		},
	}

	ctx := context.New(config.Project{
		Builds: []config.Build{
			build,
		},
	})
	err := runPipeOnBuild(ctx, build)
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(tmpDir, "pre-hook-amd64-linux"))
	assert.FileExists(t, filepath.Join(tmpDir, "post-hook-amd64-linux"))
}

func TestPipeOnBuild_hooksRunPerTarget(t *testing.T) {
	tmpDir, back := testlib.Mktmp(t)
	defer back()

	build := config.Build{
		Lang:   "fake",
		Binary: "testing.v{{.Version}}",
		Targets: []string{
			"linux_amd64",
			"darwin_amd64",
			"windows_amd64",
		},
		Hooks: config.HookConfig{
			Pre: []config.BuildHook{
				{Cmd: "touch pre-hook-{{.Target}}", Dir: tmpDir},
			},
			Post: config.BuildHooks{
				{Cmd: "touch post-hook-{{.Target}}", Dir: tmpDir},
			},
		},
	}
	ctx := context.New(config.Project{
		Builds: []config.Build{
			build,
		},
	})
	err := runPipeOnBuild(ctx, build)
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(tmpDir, "pre-hook-linux_amd64"))
	assert.FileExists(t, filepath.Join(tmpDir, "pre-hook-darwin_amd64"))
	assert.FileExists(t, filepath.Join(tmpDir, "pre-hook-windows_amd64"))
	assert.FileExists(t, filepath.Join(tmpDir, "post-hook-linux_amd64"))
	assert.FileExists(t, filepath.Join(tmpDir, "post-hook-darwin_amd64"))
	assert.FileExists(t, filepath.Join(tmpDir, "post-hook-windows_amd64"))
}

func TestPipeOnBuild_invalidBinaryTpl(t *testing.T) {
	build := config.Build{
		Lang:   "fake",
		Binary: "testing.v{{.XYZ}}",
		Targets: []string{
			"linux_amd64",
		},
	}
	ctx := context.New(config.Project{
		Builds: []config.Build{
			build,
		},
	})
	err := runPipeOnBuild(ctx, build)
	assert.EqualError(t, err, `template: tmpl:1:11: executing "tmpl" at <.XYZ>: map has no entry for key "XYZ"`)
}

func TestBuildOptionsForTarget(t *testing.T) {
	tmpDir, back := testlib.Mktmp(t)
	defer back()

	build := config.Build{
		ID:     "testid",
		Binary: "testbinary",
		Targets: []string{
			"linux_amd64",
			"darwin_amd64",
			"windows_amd64",
		},
	}
	ctx := context.New(config.Project{
		Dist:   tmpDir,
		Builds: []config.Build{build},
	})
	opts, err := buildOptionsForTarget(ctx, build, "linux_amd64")
	assert.NoError(t, err)
	assert.Equal(t, &api.Options{
		Name:   "testbinary",
		Path:   filepath.Join(tmpDir, "testid_linux_amd64", "testbinary"),
		Target: "linux_amd64",
		Os:     "linux",
		Arch:   "amd64",
	}, opts)
}

func TestHookComplex(t *testing.T) {
	tmp, back := testlib.Mktmp(t)
	defer back()

	require.NoError(t, runHook(context.New(config.Project{}), api.Options{}, []string{}, config.BuildHooks{
		{
			Cmd: `bash -c "touch foo"`,
		},
		{
			Cmd: `bash -c "touch \"bar\""`,
		},
	}))

	require.FileExists(t, filepath.Join(tmp, "foo"))
	require.FileExists(t, filepath.Join(tmp, "bar"))
}

func TestHookInvalidShelCommand(t *testing.T) {
	require.Error(t, runHook(context.New(config.Project{}), api.Options{}, []string{}, config.BuildHooks{
		{
			Cmd: `bash -c "echo \"unterminated command\"`,
		},
	}))
}
