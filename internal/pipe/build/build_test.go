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
	"github.com/stretchr/testify/require"
)

var errFailedBuild = errors.New("fake builder failed")
var errFailedDefault = errors.New("fake builder defaults failed")

type fakeBuilder struct {
	fail        bool
	failDefault bool
}

func (f *fakeBuilder) WithDefaults(build config.Build) (config.Build, error) {
	if f.failDefault {
		return build, errFailedDefault
	}
	return build, nil
}

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
	api.Register("fakeFailDefault", &fakeBuilder{
		failDefault: true,
	})
}

func TestPipeDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestBuild(t *testing.T) {
	var folder = testlib.Mktmp(t)
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
	require.NoError(t, err)
	error := doBuild(ctx, ctx.Config.Builds[0], *opts)
	require.NoError(t, error)
}

func TestRunPipe(t *testing.T) {
	var folder = testlib.Mktmp(t)
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
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, ctx.Artifacts.List(), []*artifact.Artifact{{
		Name: "testing",
	}})
}

func TestRunFullPipe(t *testing.T) {
	var folder = testlib.Mktmp(t)
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
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, ctx.Artifacts.List(), []*artifact.Artifact{{
		Name: "testing",
	}})
	require.FileExists(t, post)
	require.FileExists(t, pre)
	require.FileExists(t, filepath.Join(folder, "build1_whatever", "testing"))
}

func TestRunFullPipeFail(t *testing.T) {
	var folder = testlib.Mktmp(t)
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
	require.EqualError(t, Pipe{}.Run(ctx), errFailedBuild.Error())
	require.Empty(t, ctx.Artifacts.List())
	require.FileExists(t, pre)
}

func TestRunPipeFailingHooks(t *testing.T) {
	var folder = testlib.Mktmp(t)
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
		require.EqualError(t, Pipe{}.Run(ctx), `pre hook failed: "": exec: "exit": executable file not found in $PATH`)
	})
	t.Run("post-hook", func(t *testing.T) {
		var ctx = context.New(cfg)
		ctx.Git.CurrentTag = "2.3.4"
		ctx.Config.Builds[0].Hooks.Pre = []config.BuildHook{{Cmd: "echo pre"}}
		ctx.Config.Builds[0].Hooks.Post = []config.BuildHook{{Cmd: "exit 1"}}
		require.EqualError(t, Pipe{}.Run(ctx), `post hook failed: "": exec: "exit": executable file not found in $PATH`)
	})
}

func TestDefaultNoBuilds(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{},
	}
	require.NoError(t, Pipe{}.Default(ctx))
}

func TestDefaultFail(t *testing.T) {
	var folder = testlib.Mktmp(t)
	var config = config.Project{
		Dist: folder,
		Builds: []config.Build{
			{
				Lang: "fakeFailDefault",
			},
		},
	}
	var ctx = context.New(config)
	require.EqualError(t, Pipe{}.Default(ctx), errFailedDefault.Error())
	require.Empty(t, ctx.Artifacts.List())
}

func TestDefaultExpandEnv(t *testing.T) {
	require.NoError(t, os.Setenv("XBAR", "FOOBAR"))
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
	require.NoError(t, Pipe{}.Default(ctx))
	var env = ctx.Config.Builds[0].Env[0]
	require.Equal(t, "XFOO=bar_FOOBAR", env)
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
	require.NoError(t, Pipe{}.Default(ctx))
	var build = ctx.Config.Builds[0]
	require.Equal(t, ctx.Config.ProjectName, build.ID)
	require.Equal(t, ctx.Config.ProjectName, build.Binary)
	require.Equal(t, ".", build.Dir)
	require.Equal(t, ".", build.Main)
	require.Equal(t, []string{"linux", "darwin"}, build.Goos)
	require.Equal(t, []string{"amd64", "arm64", "386"}, build.Goarch)
	require.Equal(t, []string{"6"}, build.Goarm)
	require.Len(t, build.Ldflags, 1)
	require.Equal(t, "-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}} -X main.builtBy=goreleaser", build.Ldflags[0])
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
	require.EqualError(t, Pipe{}.Default(ctx), "found 2 builds with the ID 'foo', please fix your config")
	var build = ctx.Config.Builds[0]
	require.Equal(t, ctx.Config.ProjectName, build.ID)
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
	require.EqualError(t, Pipe{}.Default(ctx), "found 2 builds with the ID 'a', please fix your config")
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
	require.NoError(t, Pipe{}.Default(ctx))
	t.Run("build0", func(t *testing.T) {
		var build = ctx.Config.Builds[0]
		require.Equal(t, "bar", build.Binary)
		require.Equal(t, ".", build.Dir)
		require.Equal(t, "./cmd/main.go", build.Main)
		require.Equal(t, []string{"linux"}, build.Goos)
		require.Equal(t, []string{"amd64", "arm64", "386"}, build.Goarch)
		require.Equal(t, []string{"6"}, build.Goarm)
		require.Len(t, build.Ldflags, 1)
		require.Equal(t, "-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}} -X main.builtBy=goreleaser", build.Ldflags[0])
	})
	t.Run("build1", func(t *testing.T) {
		var build = ctx.Config.Builds[1]
		require.Equal(t, "foo", build.Binary)
		require.Equal(t, ".", build.Main)
		require.Equal(t, "baz", build.Dir)
		require.Equal(t, []string{"linux", "darwin"}, build.Goos)
		require.Equal(t, []string{"386"}, build.Goarch)
		require.Equal(t, []string{"6"}, build.Goarm)
		require.Len(t, build.Ldflags, 1)
		require.Equal(t, "-s -w", build.Ldflags[0])
	})
}

func TestDefaultFillSingleBuild(t *testing.T) {
	testlib.Mktmp(t)

	var ctx = &context.Context{
		Config: config.Project{
			ProjectName: "foo",
			SingleBuild: config.Build{
				Main: "testreleaser",
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Builds, 1)
	require.Equal(t, ctx.Config.Builds[0].Binary, "foo")
}

func TestDefaultFailSingleBuild(t *testing.T) {
	var folder = testlib.Mktmp(t)
	var config = config.Project{
		Dist: folder,
		SingleBuild: config.Build{
			Lang: "fakeFailDefault",
		},
	}
	var ctx = context.New(config)
	require.EqualError(t, Pipe{}.Default(ctx), errFailedDefault.Error())
	require.Empty(t, ctx.Artifacts.List())
}

func TestSkipBuild(t *testing.T) {
	var folder = testlib.Mktmp(t)
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
	require.NoError(t, Pipe{}.Run(ctx))
	require.Len(t, ctx.Artifacts.List(), 0)
}

func TestExtWindows(t *testing.T) {
	require.Equal(t, ".exe", extFor("windows_amd64", config.FlagArray{}))
	require.Equal(t, ".exe", extFor("windows_386", config.FlagArray{}))
	require.Equal(t, ".exe", extFor("windows_amd64", config.FlagArray{"-tags=dev", "-v"}))
	require.Equal(t, ".dll", extFor("windows_amd64", config.FlagArray{"-tags=dev", "-v", "-buildmode=c-shared"}))
	require.Equal(t, ".dll", extFor("windows_386", config.FlagArray{"-buildmode=c-shared"}))
	require.Equal(t, ".lib", extFor("windows_amd64", config.FlagArray{"-buildmode=c-archive"}))
	require.Equal(t, ".lib", extFor("windows_386", config.FlagArray{"-tags=dev", "-v", "-buildmode=c-archive"}))
}

func TestExtWasm(t *testing.T) {
	require.Equal(t, ".wasm", extFor("js_wasm", config.FlagArray{}))
}

func TestExtOthers(t *testing.T) {
	require.Empty(t, "", extFor("linux_amd64", config.FlagArray{}))
	require.Empty(t, "", extFor("linuxwin_386", config.FlagArray{}))
	require.Empty(t, "", extFor("winasdasd_sad", config.FlagArray{}))
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
	require.NoError(t, err)
	require.Contains(t, binary, "-s -w")
	require.Contains(t, binary, "-X main.version=1.2.3")
	require.Contains(t, binary, "-X main.tag=v1.2.3")
	require.Contains(t, binary, "-X main.commit=123")
	require.Contains(t, binary, "-X main.date=")
	require.Contains(t, binary, `-X "main.foo=123"`)
}

func TestRunHookEnvs(t *testing.T) {
	var tmp = testlib.Mktmp(t)

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
		require.NoError(t, err)
		require.FileExists(t, filepath.Join(tmp, "foo"))
	})

	t.Run("valid cmd template with build env", func(t *testing.T) {
		var err = runHook(context.New(config.Project{
			Builds: []config.Build{
				build,
			},
		}), opts, build.Env, simpleHook("touch {{ .Env.FOO }}"))
		require.NoError(t, err)
		require.FileExists(t, filepath.Join(tmp, "foo"))
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
		require.NoError(t, err)
		require.FileExists(t, filepath.Join(tmp, "hook_only"))
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
		require.NoError(t, err)

		require.FileExists(t, filepath.Join(tmp, "build_over_ctx"))
		require.NoFileExists(t, filepath.Join(tmp, "ctx_over_build"))
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
		require.NoError(t, err)
		require.FileExists(t, filepath.Join(tmp, "hook_over_ctx"))
		require.NoFileExists(t, filepath.Join(tmp, "ctx_over_hook"))
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
		require.NoError(t, err)
		require.FileExists(t, filepath.Join(tmp, "hook_over_build"))
		require.NoFileExists(t, filepath.Join(tmp, "build_over_hook"))
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
		require.NoError(t, err)
		require.FileExists(t, filepath.Join(tmp, "hook_wins"))
		require.NoFileExists(t, filepath.Join(tmp, "ctx_wins"))
		require.NoFileExists(t, filepath.Join(tmp, "build_wins"))
	})

	t.Run("invalid cmd template", func(t *testing.T) {
		var err = runHook(context.New(config.Project{
			Builds: []config.Build{
				build,
			},
		}), opts, build.Env, simpleHook("touch {{ .Env.FOOss }}"))
		require.EqualError(t, err, `template: tmpl:1:13: executing "tmpl" at <.Env.FOOss>: map has no entry for key "FOOss"`)
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
		require.EqualError(t, err, `template: tmpl:1:7: executing "tmpl" at <.Env.INVALID_ENV>: map has no entry for key "INVALID_ENV"`)
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
		require.EqualError(t, err, `template: tmpl:1:12: executing "tmpl" at <.Env.MISSING_ENV>: map has no entry for key "MISSING_ENV"`)
	})

	t.Run("build env inside shell", func(t *testing.T) {
		var shell = `#!/bin/sh -e
touch "$BAR"`
		err := ioutil.WriteFile(filepath.Join(tmp, "test.sh"), []byte(shell), 0750)
		require.NoError(t, err)
		err = runHook(context.New(config.Project{
			Builds: []config.Build{
				build,
			},
		}), opts, build.Env, simpleHook("sh test.sh"))
		require.NoError(t, err)
		require.FileExists(t, filepath.Join(tmp, "bar"))
	})
}

func TestBuild_hooksKnowGoosGoarch(t *testing.T) {
	var tmpDir = testlib.Mktmp(t)
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
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(tmpDir, "pre-hook-amd64-linux"))
	require.FileExists(t, filepath.Join(tmpDir, "post-hook-amd64-linux"))
}

func TestPipeOnBuild_hooksRunPerTarget(t *testing.T) {
	var tmpDir = testlib.Mktmp(t)

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
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(tmpDir, "pre-hook-linux_amd64"))
	require.FileExists(t, filepath.Join(tmpDir, "pre-hook-darwin_amd64"))
	require.FileExists(t, filepath.Join(tmpDir, "pre-hook-windows_amd64"))
	require.FileExists(t, filepath.Join(tmpDir, "post-hook-linux_amd64"))
	require.FileExists(t, filepath.Join(tmpDir, "post-hook-darwin_amd64"))
	require.FileExists(t, filepath.Join(tmpDir, "post-hook-windows_amd64"))
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
	require.EqualError(t, err, `template: tmpl:1:11: executing "tmpl" at <.XYZ>: map has no entry for key "XYZ"`)
}

func TestBuildOptionsForTarget(t *testing.T) {
	var tmpDir = testlib.Mktmp(t)

	testCases := []struct {
		name         string
		build        config.Build
		expectedOpts *api.Options
	}{
		{
			name: "simple options for target",
			build: config.Build{
				ID:     "testid",
				Binary: "testbinary",
				Targets: []string{
					"linux_amd64",
				},
			},
			expectedOpts: &api.Options{
				Name:   "testbinary",
				Path:   filepath.Join(tmpDir, "testid_linux_amd64", "testbinary"),
				Target: "linux_amd64",
				Os:     "linux",
				Arch:   "amd64",
			},
		},
		{
			name: "binary name with Os and Arch template variables",
			build: config.Build{
				ID:     "testid",
				Binary: "testbinary_{{.Os}}_{{.Arch}}",
				Targets: []string{
					"linux_amd64",
				},
			},
			expectedOpts: &api.Options{
				Name:   "testbinary_linux_amd64",
				Path:   filepath.Join(tmpDir, "testid_linux_amd64", "testbinary_linux_amd64"),
				Target: "linux_amd64",
				Os:     "linux",
				Arch:   "amd64",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.New(config.Project{
				Dist:   tmpDir,
				Builds: []config.Build{tc.build},
			})
			opts, err := buildOptionsForTarget(ctx, tc.build, tc.build.Targets[0])
			require.NoError(t, err)
			require.Equal(t, tc.expectedOpts, opts)
		})
	}
}

func TestHookComplex(t *testing.T) {
	var tmp = testlib.Mktmp(t)

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

func TestRunHookFailWithLogs(t *testing.T) {
	var folder = testlib.Mktmp(t)
	var config = config.Project{
		Dist: folder,
		Builds: []config.Build{
			{
				Lang:   "fakeFail",
				Binary: "testing",
				Flags:  []string{"-v"},
				Hooks: config.HookConfig{
					Pre: []config.BuildHook{
						{Cmd: "sh -c 'echo foo; exit 1'"},
					},
				},
				Targets: []string{"whatever"},
			},
		},
	}
	var ctx = context.New(config)
	ctx.Git.CurrentTag = "2.4.5"
	require.EqualError(t, Pipe{}.Run(ctx), "pre hook failed: \"foo\\n\": exit status 1")
	require.Empty(t, ctx.Artifacts.List())
}
