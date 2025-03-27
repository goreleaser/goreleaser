package build

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	api "github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

var (
	errFailedBuild   = errors.New("fake builder failed")
	errFailedDefault = errors.New("fake builder defaults failed")
)

type fakeTarget struct {
	target string
}

// String implements build.Target.
func (f fakeTarget) String() string {
	return f.target
}

// Fields implements build.Target.
func (f fakeTarget) Fields() map[string]string {
	os, arch, _ := strings.Cut(f.target, "_")
	return map[string]string{
		tmpl.KeyOS:   os,
		tmpl.KeyArch: arch,
	}
}

type fakeBuilder struct {
	fail        bool
	failDefault bool
}

// Parse implements build.Builder.
func (f *fakeBuilder) Parse(target string) (api.Target, error) {
	return fakeTarget{target}, nil
}

func (f *fakeBuilder) WithDefaults(build config.Build) (config.Build, error) {
	if f.failDefault {
		return build, errFailedDefault
	}
	return build, nil
}

func (f *fakeBuilder) Build(ctx *context.Context, _ config.Build, options api.Options) error {
	if f.fail {
		return errFailedBuild
	}
	if err := os.WriteFile(options.Path, []byte("foo"), 0o755); err != nil {
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
	folder := testlib.Mktmp(t)
	config := config.Project{
		Dist: folder,
		Builds: []config.Build{
			{
				Builder: "fake",
				Binary:  "testing.v{{.Version}}",
				BuildDetails: config.BuildDetails{
					Flags: []string{"-n"},
					Env:   []string{"BLAH=1"},
				},
			},
		},
	}

	ctx := testctx.NewWithCfg(
		config,
		testctx.WithVersion("1.2.3"),
		testctx.WithGitInfo(context.GitInfo{
			CurrentTag: "v1.2.3",
			Commit:     "123",
		}),
	)
	require.NoError(t, buildTarget(ctx, ctx.Config.Builds[0], "darwin_amd64"))
}

func TestRunPipe(t *testing.T) {
	folder := testlib.Mktmp(t)
	ctx := testctx.NewWithCfg(config.Project{
		Dist: folder,
		Builds: []config.Build{
			{
				Builder: "fake",
				Binary:  "testing",
				BuildDetails: config.BuildDetails{
					Flags:   []string{"-v"},
					Ldflags: []string{"-X main.test=testing"},
				},
				Targets: []string{"linux_amd64"},
			},
		},
	}, testctx.WithCurrentTag("2.4.5"))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, []*artifact.Artifact{{
		Name: "testing",
	}}, ctx.Artifacts.List())
}

func TestRunFullPipe(t *testing.T) {
	folder := testlib.Mktmp(t)
	pre := filepath.Join(folder, "pre")
	post := filepath.Join(folder, "post")
	preOS := filepath.Join(folder, "pre_linux")
	postOS := filepath.Join(folder, "post_linux")
	config := config.Project{
		Builds: []config.Build{
			{
				ID:      "build1",
				Builder: "fake",
				Binary:  "testing",
				BuildDetails: config.BuildDetails{
					Flags:   []string{"-v"},
					Ldflags: []string{"-X main.test=testing"},
					Env:     []string{"THE_OS={{ .Os }}"},
				},
				Hooks: config.BuildHookConfig{
					Pre: []config.Hook{
						{Cmd: testlib.Touch(pre)},
						{Cmd: testlib.Touch("pre_{{ .Env.THE_OS}}")},
					},
					Post: []config.Hook{
						{Cmd: testlib.Touch(post)},
						{Cmd: testlib.Touch("post_{{ .Env.THE_OS}}")},
					},
				},
				Targets: []string{"linux_amd64"},
			},
		},
		Dist: folder,
	}
	ctx := testctx.NewWithCfg(config, testctx.WithCurrentTag("2.4.5"))
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, []*artifact.Artifact{{
		Name: "testing",
	}}, ctx.Artifacts.List())
	require.FileExists(t, post)
	require.FileExists(t, pre)
	require.FileExists(t, postOS)
	require.FileExists(t, preOS)
	require.FileExists(t, filepath.Join(folder, "build1_linux_amd64", "testing"))
}

func TestRunFullPipeFail(t *testing.T) {
	folder := testlib.Mktmp(t)
	pre := filepath.Join(folder, "pre")
	post := filepath.Join(folder, "post")
	config := config.Project{
		Dist: folder,
		Builds: []config.Build{
			{
				Builder: "fakeFail",
				Binary:  "testing",
				BuildDetails: config.BuildDetails{
					Flags:   []string{"-v"},
					Ldflags: []string{"-X main.test=testing"},
				},
				Hooks: config.BuildHookConfig{
					Pre: []config.Hook{
						{Cmd: testlib.Touch(pre)},
					},
					Post: []config.Hook{
						{Cmd: testlib.Touch(post)},
					},
				},
				Targets: []string{"linux_amd64"},
			},
		},
	}
	ctx := testctx.NewWithCfg(config, testctx.WithCurrentTag("2.4.5"))
	require.ErrorIs(t, Pipe{}.Run(ctx), errFailedBuild)
	require.Empty(t, ctx.Artifacts.List())
	require.FileExists(t, pre)
}

func TestRunPipeFailingHooks(t *testing.T) {
	folder := testlib.Mktmp(t)
	cfg := config.Project{
		Dist: folder,
		Builds: []config.Build{
			{
				Builder: "fake",
				Binary:  "hooks",
				Hooks:   config.BuildHookConfig{},
				Targets: []string{"linux_amd64"},
			},
		},
	}
	t.Run("pre-hook", func(t *testing.T) {
		ctx := testctx.NewWithCfg(cfg, testctx.WithCurrentTag("2.4.5"))
		ctx.Config.Builds[0].Hooks.Pre = []config.Hook{{Cmd: "exit 1"}}
		ctx.Config.Builds[0].Hooks.Post = []config.Hook{{Cmd: testlib.Echo("post")}}

		err := Pipe{}.Run(ctx)
		require.ErrorIs(t, err, exec.ErrNotFound)
		require.ErrorContains(t, err, "pre hook failed")
	})
	t.Run("post-hook", func(t *testing.T) {
		ctx := testctx.NewWithCfg(cfg, testctx.WithCurrentTag("2.4.5"))
		ctx.Config.Builds[0].Hooks.Pre = []config.Hook{{Cmd: testlib.Echo("pre")}}
		ctx.Config.Builds[0].Hooks.Post = []config.Hook{{Cmd: "exit 1"}}
		err := Pipe{}.Run(ctx)
		require.ErrorIs(t, err, exec.ErrNotFound)
		require.ErrorContains(t, err, "post hook failed")
	})

	t.Run("post-hook-skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(
			cfg,
			testctx.WithCurrentTag("2.4.5"),
			testctx.Skip(skips.PostBuildHooks),
		)
		ctx.Config.Builds[0].Hooks.Pre = []config.Hook{{Cmd: testlib.Echo("pre")}}
		ctx.Config.Builds[0].Hooks.Post = []config.Hook{{Cmd: "exit 1"}}
		require.NoError(t, Pipe{}.Run(ctx))
	})

	t.Run("pre-hook-skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(
			cfg,
			testctx.WithCurrentTag("2.4.5"),
			testctx.Skip(skips.PreBuildHooks),
		)
		ctx.Config.Builds[0].Hooks.Pre = []config.Hook{{Cmd: "exit 1"}}
		ctx.Config.Builds[0].Hooks.Post = []config.Hook{{Cmd: testlib.Echo("pre")}}
		require.NoError(t, Pipe{}.Run(ctx))
	})
}

func TestDefaultNoBuilds(t *testing.T) {
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Default(ctx))
}

func TestDefaultFail(t *testing.T) {
	folder := testlib.Mktmp(t)
	config := config.Project{
		Dist: folder,
		Builds: []config.Build{
			{
				Builder: "fakeFailDefault",
			},
		},
	}
	ctx := testctx.NewWithCfg(config)
	require.EqualError(t, Pipe{}.Default(ctx), errFailedDefault.Error())
	require.Empty(t, ctx.Artifacts.List())
}

func TestDefaultExpandEnv(t *testing.T) {
	t.Setenv("XBAR", "FOOBAR")
	ctx := testctx.NewWithCfg(config.Project{
		Builds: []config.Build{
			{
				BuildDetails: config.BuildDetails{
					Env: []string{
						"XFOO=bar_$XBAR",
					},
				},
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	env := ctx.Config.Builds[0].Env[0]
	require.Equal(t, "XFOO=bar_FOOBAR", env)
}

func TestDefaultEmptyBuild(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "foo",
		Builds: []config.Build{
			{},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	build := ctx.Config.Builds[0]
	require.Equal(t, ctx.Config.ProjectName, build.ID)
	require.Equal(t, ctx.Config.ProjectName, build.Binary)
	require.Equal(t, ".", build.Dir)
	require.Equal(t, ".", build.Main)
	require.Equal(t, []string{"linux", "darwin", "windows"}, build.Goos)
	require.Equal(t, []string{"amd64", "arm64", "386"}, build.Goarch)
	require.Equal(t, []string{"6"}, build.Goarm)
	require.Equal(t, []string{"hardfloat"}, build.Gomips)
	require.Equal(t, []string{"v1"}, build.Goamd64)
	require.Len(t, build.Ldflags, 1)
	require.Equal(t, "-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}} -X main.builtBy=goreleaser", build.Ldflags[0])
}

func TestDefaultBuildID(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "foo",
		Builds: []config.Build{
			{
				Binary: "{{.Env.FOO}}",
			},
			{
				Binary: "bar",
			},
		},
	})
	require.EqualError(t, Pipe{}.Default(ctx), "found 2 builds with the ID 'foo', please fix your config")
	build1 := ctx.Config.Builds[0].ID
	build2 := ctx.Config.Builds[1].ID
	require.Equal(t, build1, build2)
	require.Equal(t, "foo", build2)
}

func TestSeveralBuildsWithTheSameID(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
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
	})
	require.EqualError(t, Pipe{}.Default(ctx), "found 2 builds with the ID 'a', please fix your config")
}

func TestDefaultPartialBuilds(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Builds: []config.Build{
			{
				ID:     "build1",
				Binary: "bar",
				Goos:   []string{"linux"},
				Main:   "./cmd/main.go",
			},
			{
				ID:     "build2",
				Binary: "foo",
				Dir:    "baz",
				BuildDetails: config.BuildDetails{
					Ldflags: []string{"-s -w"},
				},
				Goarch: []string{"386"},
			},
		},
	})
	// Create any 'Dir' paths necessary for builds.
	t.Chdir(t.TempDir())
	for _, b := range ctx.Config.Builds {
		if b.Dir != "" {
			require.NoError(t, os.Mkdir(b.Dir, 0o755))
		}
	}
	require.NoError(t, Pipe{}.Default(ctx))

	t.Run("build0", func(t *testing.T) {
		build := ctx.Config.Builds[0]
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
		build := ctx.Config.Builds[1]
		require.Equal(t, "foo", build.Binary)
		require.Equal(t, ".", build.Main)
		require.Equal(t, "baz", build.Dir)
		require.Equal(t, []string{"linux", "darwin", "windows"}, build.Goos)
		require.Equal(t, []string{"386"}, build.Goarch)
		require.Equal(t, []string{"6"}, build.Goarm)
		require.Len(t, build.Ldflags, 1)
		require.Equal(t, "-s -w", build.Ldflags[0])
	})
}

func TestSkipBuild(t *testing.T) {
	folder := testlib.Mktmp(t)
	config := config.Project{
		Dist: folder,
		Builds: []config.Build{
			{
				Skip: "true",
			},
		},
	}
	ctx := testctx.NewWithCfg(config, testctx.WithCurrentTag("2.4.5"))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Empty(t, ctx.Artifacts.List())
}

func TestSkipBuildTmpl(t *testing.T) {
	folder := testlib.Mktmp(t)
	config := config.Project{
		Dist: folder,
		Env:  []string{"FOO=bar"},
		Builds: []config.Build{
			{
				Skip: "{{ eq .Env.FOO \"bar\" }}",
			},
		},
	}
	ctx := testctx.NewWithCfg(config, testctx.WithCurrentTag("2.4.5"))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Empty(t, ctx.Artifacts.List())
}

func TestExtDarwin(t *testing.T) {
	require.Empty(t, extFor("darwin_amd64", config.BuildDetails{}))
	require.Empty(t, extFor("darwin_arm64", config.BuildDetails{}))
	require.Empty(t, extFor("darwin_amd64", config.BuildDetails{}))
	require.Equal(t, ".dylib", extFor("darwin_amd64", config.BuildDetails{Buildmode: "c-shared"}))
	require.Equal(t, ".dylib", extFor("darwin_arm64", config.BuildDetails{Buildmode: "c-shared"}))
	require.Equal(t, ".a", extFor("darwin_amd64", config.BuildDetails{Buildmode: "c-archive"}))
	require.Equal(t, ".a", extFor("darwin_arm64", config.BuildDetails{Buildmode: "c-archive"}))
}

func TestExtLinux(t *testing.T) {
	require.Empty(t, extFor("linux_amd64", config.BuildDetails{}))
	require.Empty(t, extFor("linux_386", config.BuildDetails{}))
	require.Empty(t, extFor("linux_amd64", config.BuildDetails{}))
	require.Equal(t, ".so", extFor("linux_amd64", config.BuildDetails{Buildmode: "c-shared"}))
	require.Equal(t, ".so", extFor("linux_386", config.BuildDetails{Buildmode: "c-shared"}))
	require.Equal(t, ".a", extFor("linux_amd64", config.BuildDetails{Buildmode: "c-archive"}))
	require.Equal(t, ".a", extFor("linux_386", config.BuildDetails{Buildmode: "c-archive"}))
}

func TestExtWindows(t *testing.T) {
	require.Equal(t, ".exe", extFor("windows_amd64", config.BuildDetails{}))
	require.Equal(t, ".exe", extFor("windows_386", config.BuildDetails{}))
	require.Equal(t, ".exe", extFor("windows_amd64", config.BuildDetails{}))
	require.Equal(t, ".dll", extFor("windows_amd64", config.BuildDetails{Buildmode: "c-shared"}))
	require.Equal(t, ".dll", extFor("windows_386", config.BuildDetails{Buildmode: "c-shared"}))
	require.Equal(t, ".lib", extFor("windows_amd64", config.BuildDetails{Buildmode: "c-archive"}))
	require.Equal(t, ".lib", extFor("windows_386", config.BuildDetails{Buildmode: "c-archive"}))
}

func TestExtWasm(t *testing.T) {
	require.Equal(t, ".wasm", extFor("js_wasm", config.BuildDetails{}))
	require.Equal(t, ".wasm", extFor("wasip1_wasm", config.BuildDetails{}))
	require.Equal(t, ".wasm", extFor("wasip1_wasm", config.BuildDetails{Buildmode: "c-shared"}))
}

func TestExtOthers(t *testing.T) {
	require.Empty(t, extFor("linux_amd64", config.BuildDetails{}))
	require.Empty(t, extFor("linuxwin_386", config.BuildDetails{}))
	require.Empty(t, extFor("winasdasd_sad", config.BuildDetails{}))
	require.Equal(t, ".so", extFor("aix_amd64", config.BuildDetails{Buildmode: "c-shared"}))
	require.Equal(t, ".a", extFor("android_386", config.BuildDetails{Buildmode: "c-archive"}))
	require.Equal(t, ".so", extFor("winasdasd_sad", config.BuildDetails{Buildmode: "c-shared"}))
}

func TestTemplate(t *testing.T) {
	ctx := testctx.New(
		testctx.WithEnv(map[string]string{"FOO": "123"}),
		testctx.WithVersion("1.2.3"),
		testctx.WithCurrentTag("v1.2.3"),
		testctx.WithCommit("123"),
	)
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

func TestBuild_hooksKnowGoosGoarch(t *testing.T) {
	tmpDir := testlib.Mktmp(t)
	build := config.Build{
		Builder: "fake",
		Goarch:  []string{"amd64"},
		Goos:    []string{"linux"},
		Binary:  "testing-goos-goarch.v{{.Version}}",
		Targets: []string{
			"linux_amd64",
		},
		Hooks: config.BuildHookConfig{
			Pre: []config.Hook{
				{Cmd: testlib.Touch("pre-hook-{{.Arch}}-{{.Os}}"), Dir: tmpDir},
			},
			Post: config.Hooks{
				{Cmd: testlib.Touch(" post-hook-{{.Arch}}-{{.Os}}"), Dir: tmpDir},
			},
		},
	}

	ctx := testctx.NewWithCfg(config.Project{
		Builds: []config.Build{
			build,
		},
	})
	g := semerrgroup.New(ctx.Parallelism)
	runPipeOnBuild(ctx, g, build)
	require.NoError(t, g.Wait())
	require.FileExists(t, filepath.Join(tmpDir, "pre-hook-amd64-linux"))
	require.FileExists(t, filepath.Join(tmpDir, "post-hook-amd64-linux"))
}

func TestPipeOnBuild_hooksRunPerTarget(t *testing.T) {
	tmpDir := testlib.Mktmp(t)

	build := config.Build{
		Builder: "fake",
		Binary:  "testing.v{{.Version}}",
		Targets: []string{
			"linux_amd64",
			"darwin_amd64",
			"windows_amd64",
		},
		Hooks: config.BuildHookConfig{
			Pre: []config.Hook{
				{Cmd: testlib.Touch("pre-hook-{{.Target}}"), Dir: tmpDir},
			},
			Post: config.Hooks{
				{Cmd: testlib.Touch("post-hook-{{.Target}}"), Dir: tmpDir},
			},
		},
	}
	ctx := testctx.NewWithCfg(config.Project{
		Builds: []config.Build{
			build,
		},
	})
	g := semerrgroup.New(ctx.Parallelism)
	runPipeOnBuild(ctx, g, build)
	require.NoError(t, g.Wait())
	require.FileExists(t, filepath.Join(tmpDir, "pre-hook-linux_amd64"))
	require.FileExists(t, filepath.Join(tmpDir, "pre-hook-darwin_amd64"))
	require.FileExists(t, filepath.Join(tmpDir, "pre-hook-windows_amd64"))
	require.FileExists(t, filepath.Join(tmpDir, "post-hook-linux_amd64"))
	require.FileExists(t, filepath.Join(tmpDir, "post-hook-darwin_amd64"))
	require.FileExists(t, filepath.Join(tmpDir, "post-hook-windows_amd64"))
}

func TestPipeOnBuild_invalidBinaryTpl(t *testing.T) {
	build := config.Build{
		Builder: "fake",
		Binary:  "testing.v{{.XYZ}}",
		Targets: []string{
			"linux_amd64",
		},
	}
	ctx := testctx.NewWithCfg(config.Project{
		Builds: []config.Build{
			build,
		},
	})
	g := semerrgroup.New(ctx.Parallelism)
	runPipeOnBuild(ctx, g, build)
	testlib.RequireTemplateError(t, g.Wait())
}

func TestBuildOptionsForTarget(t *testing.T) {
	tmpDir := testlib.Mktmp(t)

	testCases := []struct {
		name         string
		build        config.Build
		expectedOpts api.Options
		expectedErr  string
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
			expectedOpts: api.Options{
				Name: "testbinary",
				Path: filepath.Join(tmpDir, "testid_linux_amd64_v1", "testbinary"),
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
			expectedOpts: api.Options{
				Name: "testbinary_linux_amd64",
				Path: filepath.Join(tmpDir, "testid_linux_amd64_v1", "testbinary_linux_amd64"),
			},
		},
		{
			name: "no unique dist path evals true",
			build: config.Build{
				ID:     "testid",
				Binary: "distpath/{{.Os}}/{{.Arch}}/testbinary",
				Targets: []string{
					"linux_amd64",
				},
				NoUniqueDistDir: `{{ printf "true"}}`,
			},
			expectedOpts: api.Options{
				Name: "distpath/linux/amd64/testbinary",
				Path: filepath.Join(tmpDir, "distpath", "linux", "amd64", "testbinary"),
			},
		},
		{
			name: "no unique dist path evals false",
			build: config.Build{
				ID:     "testid",
				Binary: "testbinary",
				Targets: []string{
					"linux_amd64",
				},
				NoUniqueDistDir: `{{ printf "false"}}`,
			},
			expectedOpts: api.Options{
				Name: "testbinary",
				Path: filepath.Join(tmpDir, "testid_linux_amd64_v1", "testbinary"),
			},
		},
		{
			name: "with goarm",
			build: config.Build{
				ID:     "testid",
				Binary: "testbinary",
				Targets: []string{
					"linux_arm_6",
				},
			},
			expectedOpts: api.Options{
				Name: "testbinary",
				Path: filepath.Join(tmpDir, "testid_linux_arm_6", "testbinary"),
			},
		},
		{
			name: "with gomips",
			build: config.Build{
				ID:     "testid",
				Binary: "testbinary",
				Targets: []string{
					"linux_mips_softfloat",
				},
			},
			expectedOpts: api.Options{
				Name: "testbinary",
				Path: filepath.Join(tmpDir, "testid_linux_mips_softfloat", "testbinary"),
			},
		},
		{
			name: "with goamd64",
			build: config.Build{
				ID:     "testid",
				Binary: "testbinary",
				Targets: []string{
					"linux_amd64_v3",
				},
			},
			expectedOpts: api.Options{
				Name: "testbinary",
				Path: filepath.Join(tmpDir, "testid_linux_amd64_v3", "testbinary"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := testctx.NewWithCfg(config.Project{
				Dist:   tmpDir,
				Builds: []config.Build{tc.build},
			})
			require.NoError(t, Pipe{}.Default(ctx))
			opts, err := buildOptionsForTarget(ctx, ctx.Config.Builds[0], ctx.Config.Builds[0].Targets[0])
			if tc.expectedErr == "" {
				require.NoError(t, err)
				opts.Target = nil
				require.Equal(t, tc.expectedOpts, *opts)
			} else {
				require.EqualError(t, err, tc.expectedErr)
			}
		})
	}
}

func TestRunHookFailWithLogs(t *testing.T) {
	testlib.SkipIfWindows(t, "subshells don't work in windows")
	folder := testlib.Mktmp(t)
	config := config.Project{
		Dist: folder,
		Builds: []config.Build{
			{
				Builder: "fakeFail",
				Binary:  "testing",
				BuildDetails: config.BuildDetails{
					Flags: []string{"-v"},
				},
				Hooks: config.BuildHookConfig{
					Pre: []config.Hook{
						{Cmd: "sh -c 'echo foo; exit 1'"},
					},
				},
				Targets: []string{"linux_amd64"},
			},
		},
	}
	ctx := testctx.NewWithCfg(config, testctx.WithCurrentTag("2.4.5"))
	err := Pipe{}.Run(ctx)
	require.ErrorContains(t, err, "pre hook failed")
	require.Empty(t, ctx.Artifacts.List())
}
