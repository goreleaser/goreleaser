package golang

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	api "github.com/goreleaser/goreleaser/pkg/build"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

var runtimeTarget = runtime.GOOS + "_" + runtime.GOARCH

func TestWithDefaults(t *testing.T) {
	for name, testcase := range map[string]struct {
		build    config.Build
		targets  []string
		goBinary string
	}{
		"full": {
			build: config.Build{
				ID:     "foo",
				Binary: "foo",
				Goos: []string{
					"linux",
					"windows",
					"darwin",
				},
				Goarch: []string{
					"amd64",
					"arm",
					"mips",
				},
				Goarm: []string{
					"6",
				},
				Gomips: []string{
					"softfloat",
				},
				GoBinary: "go1.2.3",
			},
			targets: []string{
				"linux_amd64",
				"linux_mips_softfloat",
				"darwin_amd64",
				"windows_amd64",
				"windows_arm_6",
				"linux_arm_6",
			},
			goBinary: "go1.2.3",
		},
		"empty": {
			build: config.Build{
				ID:     "foo2",
				Binary: "foo",
			},
			targets: []string{
				"linux_amd64",
				"linux_386",
				"linux_arm64",
				"darwin_amd64",
				"darwin_arm64",
			},
			goBinary: "go",
		},
		"custom targets": {
			build: config.Build{
				ID:     "foo3",
				Binary: "foo",
				Targets: []string{
					"linux_386",
					"darwin_amd64",
				},
			},
			targets: []string{
				"linux_386",
				"darwin_amd64",
			},
			goBinary: "go",
		},
		"empty with custom dir": {
			build: config.Build{
				ID:     "foo2",
				Binary: "foo",
				Dir:    "./testdata",
			},
			targets: []string{
				"linux_amd64",
				"linux_386",
				"linux_arm64",
				"darwin_amd64",
				"darwin_arm64",
			},
			goBinary: "go",
		},
		"empty with custom dir that doest exist": {
			build: config.Build{
				ID:     "foo2",
				Binary: "foo",
				Dir:    "./nope",
			},
			targets: []string{
				"linux_amd64",
				"linux_386",
				"linux_arm64",
				"darwin_amd64",
				"darwin_arm64",
			},
			goBinary: "go",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if testcase.build.GoBinary != "" && testcase.build.GoBinary != "go" {
				createFakeGoBinaryWithVersion(t, testcase.build.GoBinary, "go1.17")
			}
			config := config.Project{
				Builds: []config.Build{
					testcase.build,
				},
			}
			ctx := context.New(config)
			ctx.Git.CurrentTag = "5.6.7"
			build, err := Default.WithDefaults(ctx.Config.Builds[0])
			require.NoError(t, err)
			require.ElementsMatch(t, build.Targets, testcase.targets)
			require.EqualValues(t, testcase.goBinary, build.GoBinary)
		})
	}
}

// createFakeGoBinaryWithVersion creates a temporary executable with the
// given name, which will output a go version string with the given version.
//  The temporary directory created by this function will be placed in the PATH
// variable for the duration of (and cleaned up at the end of) the
// current test run.
func createFakeGoBinaryWithVersion(tb testing.TB, name, version string) {
	tb.Helper()
	d := tb.TempDir()

	require.NoError(tb, os.WriteFile(
		filepath.Join(d, name),
		[]byte(fmt.Sprintf("#!/bin/sh\necho %s", version)),
		0o755,
	))

	currentPath := os.Getenv("PATH")
	tb.Cleanup(func() {
		require.NoError(tb, os.Setenv("PATH", currentPath))
	})

	path := fmt.Sprintf("%s%c%s", d, os.PathListSeparator, currentPath)
	require.NoError(tb, os.Setenv("PATH", path))
}

func TestInvalidTargets(t *testing.T) {
	type testcase struct {
		build       config.Build
		expectedErr string
	}
	for s, tc := range map[string]testcase{
		"goos": {
			build: config.Build{
				Goos: []string{"darwin", "darwim"},
			},
			expectedErr: "invalid goos: darwim",
		},
		"goarch": {
			build: config.Build{
				Goarch: []string{"amd64", "i386", "386"},
			},
			expectedErr: "invalid goarch: i386",
		},
		"goarm": {
			build: config.Build{
				Goarch: []string{"arm"},
				Goarm:  []string{"6", "9", "8", "7"},
			},
			expectedErr: "invalid goarm: 9",
		},
		"gomips": {
			build: config.Build{
				Goarch: []string{"mips"},
				Gomips: []string{"softfloat", "mehfloat", "hardfloat"},
			},
			expectedErr: "invalid gomips: mehfloat",
		},
	} {
		t.Run(s, func(t *testing.T) {
			config := config.Project{
				Builds: []config.Build{
					tc.build,
				},
			}
			ctx := context.New(config)
			_, err := Default.WithDefaults(ctx.Config.Builds[0])
			require.EqualError(t, err, tc.expectedErr)
		})
	}
}

func TestBuild(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	config := config.Project{
		Builds: []config.Build{
			{
				ID:     "foo",
				Env:    []string{"GO111MODULE=off"},
				Binary: "bin/foo-{{ .Version }}",
				Targets: []string{
					"linux_amd64",
					"darwin_amd64",
					"windows_amd64",
					"linux_arm_6",
					"js_wasm",
					"linux_mips_softfloat",
					"linux_mips64le_softfloat",
				},
				Asmflags: []string{".=", "all="},
				Gcflags:  []string{"all="},
				Flags:    []string{"{{.Env.GO_FLAGS}}"},
				Tags:     []string{"osusergo", "netgo", "static_build"},
				GoBinary: "go",
			},
		},
	}
	ctx := context.New(config)
	ctx.Env["GO_FLAGS"] = "-v"
	ctx.Git.CurrentTag = "5.6.7"
	ctx.Version = "v" + ctx.Git.CurrentTag
	build := ctx.Config.Builds[0]
	for _, target := range build.Targets {
		var ext string
		if strings.HasPrefix(target, "windows") {
			ext = ".exe"
		} else if target == "js_wasm" {
			ext = ".wasm"
		}
		bin, terr := tmpl.New(ctx).Apply(build.Binary)
		require.NoError(t, terr)

		// injecting some delay here to force inconsistent mod times on bins
		time.Sleep(2 * time.Second)

		parts := strings.Split(target, "_")
		goos := parts[0]
		goarch := parts[1]
		goarm := ""
		gomips := ""
		if len(parts) > 2 {
			if strings.Contains(goarch, "arm") {
				goarm = parts[2]
			}
			if strings.Contains(goarch, "mips") {
				gomips = parts[2]
			}
		}
		err := Default.Build(ctx, build, api.Options{
			Target: target,
			Name:   bin + ext,
			Path:   filepath.Join(folder, "dist", target, bin+ext),
			Goos:   goos,
			Goarch: goarch,
			Goarm:  goarm,
			Gomips: gomips,
			Ext:    ext,
		})
		require.NoError(t, err)
	}
	require.ElementsMatch(t, ctx.Artifacts.List(), []*artifact.Artifact{
		{
			Name:   "bin/foo-v5.6.7",
			Path:   filepath.Join(folder, "dist", "linux_amd64", "bin", "foo-v5.6.7"),
			Goos:   "linux",
			Goarch: "amd64",
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				"Ext":    "",
				"Binary": "foo-v5.6.7",
				"ID":     "foo",
			},
		},
		{
			Name:   "bin/foo-v5.6.7",
			Path:   filepath.Join(folder, "dist", "linux_mips_softfloat", "bin", "foo-v5.6.7"),
			Goos:   "linux",
			Goarch: "mips",
			Gomips: "softfloat",
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				"Ext":    "",
				"Binary": "foo-v5.6.7",
				"ID":     "foo",
			},
		},
		{
			Name:   "bin/foo-v5.6.7",
			Path:   filepath.Join(folder, "dist", "linux_mips64le_softfloat", "bin", "foo-v5.6.7"),
			Goos:   "linux",
			Goarch: "mips64le",
			Gomips: "softfloat",
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				"Ext":    "",
				"Binary": "foo-v5.6.7",
				"ID":     "foo",
			},
		},
		{
			Name:   "bin/foo-v5.6.7",
			Path:   filepath.Join(folder, "dist", "darwin_amd64", "bin", "foo-v5.6.7"),
			Goos:   "darwin",
			Goarch: "amd64",
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				"Ext":    "",
				"Binary": "foo-v5.6.7",
				"ID":     "foo",
			},
		},
		{
			Name:   "bin/foo-v5.6.7",
			Path:   filepath.Join(folder, "dist", "linux_arm_6", "bin", "foo-v5.6.7"),
			Goos:   "linux",
			Goarch: "arm",
			Goarm:  "6",
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				"Ext":    "",
				"Binary": "foo-v5.6.7",
				"ID":     "foo",
			},
		},
		{
			Name:   "bin/foo-v5.6.7.exe",
			Path:   filepath.Join(folder, "dist", "windows_amd64", "bin", "foo-v5.6.7.exe"),
			Goos:   "windows",
			Goarch: "amd64",
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				"Ext":    ".exe",
				"Binary": "foo-v5.6.7",
				"ID":     "foo",
			},
		},
		{
			Name:   "bin/foo-v5.6.7.wasm",
			Path:   filepath.Join(folder, "dist", "js_wasm", "bin", "foo-v5.6.7.wasm"),
			Goos:   "js",
			Goarch: "wasm",
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				"Ext":    ".wasm",
				"Binary": "foo-v5.6.7",
				"ID":     "foo",
			},
		},
	})

	modTimes := map[time.Time]bool{}
	for _, bin := range ctx.Artifacts.List() {
		if bin.Type != artifact.Binary {
			continue
		}

		fi, err := os.Stat(bin.Path)
		require.NoError(t, err)

		// make this a suitable map key, per docs: https://golang.org/pkg/time/#Time
		modTime := fi.ModTime().UTC().Round(0)

		if modTimes[modTime] {
			t.Fatal("duplicate modified time found, times should be different by default")
		}
		modTimes[modTime] = true
	}
}

func TestBuildCodeInSubdir(t *testing.T) {
	folder := testlib.Mktmp(t)
	subdir := filepath.Join(folder, "bar")
	err := os.Mkdir(subdir, 0o755)
	require.NoError(t, err)
	writeGoodMain(t, subdir)
	config := config.Project{
		Builds: []config.Build{
			{
				ID:     "foo",
				Env:    []string{"GO111MODULE=off"},
				Dir:    "bar",
				Binary: "foo",
				Targets: []string{
					runtimeTarget,
				},
				GoBinary: "go",
			},
		},
	}
	ctx := context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	build := ctx.Config.Builds[0]
	err = Default.Build(ctx, build, api.Options{
		Target: runtimeTarget,
		Name:   build.Binary,
		Path:   filepath.Join(folder, "dist", runtimeTarget, build.Binary),
		Ext:    "",
	})
	require.NoError(t, err)
}

func TestBuildWithDotGoDir(t *testing.T) {
	folder := testlib.Mktmp(t)
	require.NoError(t, os.Mkdir(filepath.Join(folder, ".go"), 0o755))
	writeGoodMain(t, folder)
	config := config.Project{
		Builds: []config.Build{
			{
				ID:       "foo",
				Env:      []string{"GO111MODULE=off"},
				Binary:   "foo",
				Targets:  []string{runtimeTarget},
				GoBinary: "go",
			},
		},
	}
	ctx := context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	build := ctx.Config.Builds[0]
	require.NoError(t, Default.Build(ctx, build, api.Options{
		Target: runtimeTarget,
		Name:   build.Binary,
		Path:   filepath.Join(folder, "dist", runtimeTarget, build.Binary),
		Ext:    "",
	}))
}

func TestBuildFailed(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	config := config.Project{
		Builds: []config.Build{
			{
				ID:    "buildid",
				Flags: []string{"-flag-that-dont-exists-to-force-failure"},
				Targets: []string{
					runtimeTarget,
				},
				GoBinary: "go",
			},
		},
	}
	ctx := context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	err := Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: "darwin_amd64",
	})
	assertContainsError(t, err, `flag provided but not defined: -flag-that-dont-exists-to-force-failure`)
	require.Empty(t, ctx.Artifacts.List())
}

func TestRunInvalidAsmflags(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	config := config.Project{
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
	ctx := context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	err := Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: runtimeTarget,
	})
	require.EqualError(t, err, `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunInvalidGcflags(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	config := config.Project{
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
	ctx := context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	err := Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: runtimeTarget,
	})
	require.EqualError(t, err, `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunInvalidLdflags(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	config := config.Project{
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
	ctx := context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	err := Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: runtimeTarget,
	})
	require.EqualError(t, err, `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunInvalidFlags(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	config := config.Project{
		Builds: []config.Build{
			{
				Binary: "nametest",
				Flags:  []string{"{{.Env.GOOS}"},
				Targets: []string{
					runtimeTarget,
				},
			},
		},
	}
	ctx := context.New(config)
	err := Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: runtimeTarget,
	})
	require.EqualError(t, err, `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunPipeWithoutMainFunc(t *testing.T) {
	newCtx := func(t *testing.T) *context.Context {
		t.Helper()
		folder := testlib.Mktmp(t)
		writeMainWithoutMainFunc(t, folder)
		config := config.Project{
			Builds: []config.Build{
				{
					Binary: "no-main",
					Hooks:  config.HookConfig{},
					Targets: []string{
						runtimeTarget,
					},
				},
			},
		}
		ctx := context.New(config)
		ctx.Git.CurrentTag = "5.6.7"
		return ctx
	}
	t.Run("empty", func(t *testing.T) {
		ctx := newCtx(t)
		ctx.Config.Builds[0].Main = ""
		require.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}), `build for no-main does not contain a main function`)
	})
	t.Run("not main.go", func(t *testing.T) {
		ctx := newCtx(t)
		ctx.Config.Builds[0].Main = "foo.go"
		require.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}), `couldn't find main file: stat foo.go: no such file or directory`)
	})
	t.Run("glob", func(t *testing.T) {
		ctx := newCtx(t)
		ctx.Config.Builds[0].Main = "."
		require.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}), `build for no-main does not contain a main function`)
	})
	t.Run("fixed main.go", func(t *testing.T) {
		ctx := newCtx(t)
		ctx.Config.Builds[0].Main = "main.go"
		require.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}), `build for no-main does not contain a main function`)
	})
	t.Run("using gomod.proxy", func(t *testing.T) {
		ctx := newCtx(t)
		ctx.Config.GoMod.Proxy = true
		ctx.Config.Builds[0].Dir = "dist/proxy/test"
		ctx.Config.Builds[0].Main = "github.com/caarlos0/test"
		ctx.Config.Builds[0].UnproxiedDir = "."
		ctx.Config.Builds[0].UnproxiedMain = "."
		require.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}), `build for no-main does not contain a main function`)
	})
}

func TestRunPipeWithProxiedRepo(t *testing.T) {
	folder := testlib.Mktmp(t)
	out, err := exec.Command("git", "clone", "https://github.com/goreleaser/goreleaser", "-b", "v0.161.1", "--depth=1", ".").CombinedOutput()
	require.NoError(t, err, string(out))

	proxied := filepath.Join(folder, "dist/proxy/default")
	require.NoError(t, os.MkdirAll(proxied, 0o750))
	require.NoError(t, os.WriteFile(
		filepath.Join(proxied, "main.go"),
		[]byte(`// +build main
package main

import _ "github.com/goreleaser/goreleaser"
`),
		0o666,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(proxied, "go.mod"),
		[]byte("module foo\nrequire github.com/goreleaser/goreleaser v0.161.1"),
		0o666,
	))

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = proxied
	require.NoError(t, cmd.Run())

	config := config.Project{
		GoMod: config.GoMod{
			Proxy: true,
		},
		Builds: []config.Build{
			{
				Binary:        "foo",
				Main:          "github.com/goreleaser/goreleaser",
				Dir:           proxied,
				UnproxiedMain: ".",
				UnproxiedDir:  ".",
				Targets: []string{
					runtimeTarget,
				},
				GoBinary: "go",
			},
		},
	}
	ctx := context.New(config)

	require.NoError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: runtimeTarget,
	}))
}

func TestRunPipeWithMainFuncNotInMainGoFile(t *testing.T) {
	folder := testlib.Mktmp(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "foo.go"),
		[]byte("package main\nfunc main() {println(0)}"),
		0o644,
	))
	config := config.Project{
		Builds: []config.Build{
			{
				Env:    []string{"GO111MODULE=off"},
				Binary: "foo",
				Hooks:  config.HookConfig{},
				Targets: []string{
					runtimeTarget,
				},
				GoBinary: "go",
			},
		},
	}
	ctx := context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	t.Run("empty", func(t *testing.T) {
		ctx.Config.Builds[0].Main = ""
		require.NoError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}))
	})
	t.Run("foo.go", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "foo.go"
		require.NoError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}))
	})
	t.Run("glob", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "."
		require.NoError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}))
	})
}

func TestLdFlagsFullTemplate(t *testing.T) {
	run := time.Now().UTC()
	commit := time.Now().AddDate(-1, 0, 0)

	ctx := &context.Context{
		Git: context.GitInfo{
			CurrentTag: "v1.2.3",
			Commit:     "123",
			CommitDate: commit,
		},
		Date:    run,
		Version: "1.2.3",
		Env:     map[string]string{"FOO": "123"},
	}
	artifact := &artifact.Artifact{Goarch: "amd64"}
	flags, err := tmpl.New(ctx).WithArtifact(artifact, map[string]string{}).
		Apply(`-s -w -X main.version={{.Version}} -X main.tag={{.Tag}} -X main.date={{.Date}} -X main.commit={{.Commit}} -X "main.foo={{.Env.FOO}}" -X main.time={{ time "20060102" }} -X main.arch={{.Arch}} -X main.commitDate={{.CommitDate}}`)
	require.NoError(t, err)
	require.Contains(t, flags, "-s -w")
	require.Contains(t, flags, "-X main.version=1.2.3")
	require.Contains(t, flags, "-X main.tag=v1.2.3")
	require.Contains(t, flags, "-X main.commit=123")
	require.Contains(t, flags, fmt.Sprintf("-X main.date=%d", run.Year()))
	require.Contains(t, flags, fmt.Sprintf("-X main.time=%d", run.Year()))
	require.Contains(t, flags, `-X "main.foo=123"`)
	require.Contains(t, flags, `-X main.arch=amd64`)
	require.Contains(t, flags, fmt.Sprintf("-X main.commitDate=%d", commit.Year()))
}

func TestInvalidTemplate(t *testing.T) {
	for template, eerr := range map[string]string{
		"{{ .Nope }":    `template: tmpl:1: unexpected "}" in operand`,
		"{{.Env.NOPE}}": `template: tmpl:1:6: executing "tmpl" at <.Env.NOPE>: map has no entry for key "NOPE"`,
	} {
		t.Run(template, func(t *testing.T) {
			ctx := context.New(config.Project{})
			ctx.Git.CurrentTag = "3.4.1"
			flags, err := tmpl.New(ctx).Apply(template)
			require.EqualError(t, err, eerr)
			require.Empty(t, flags)
		})
	}
}

func TestProcessFlags(t *testing.T) {
	ctx := &context.Context{
		Version: "1.2.3",
	}
	ctx.Git.CurrentTag = "5.6.7"

	artifact := &artifact.Artifact{
		Name:   "name",
		Goos:   "darwin",
		Goarch: "amd64",
		Goarm:  "7",
		Extra: map[string]interface{}{
			"Binary": "binary",
		},
	}

	source := []string{
		"flag",
		"{{.Version}}",
		"{{.Os}}",
		"{{.Arch}}",
		"{{.Arm}}",
		"{{.Binary}}",
		"{{.ArtifactName}}",
	}

	expected := []string{
		"-testflag=flag",
		"-testflag=1.2.3",
		"-testflag=darwin",
		"-testflag=amd64",
		"-testflag=7",
		"-testflag=binary",
		"-testflag=name",
	}

	flags, err := processFlags(ctx, artifact, []string{}, source, "-testflag=")
	require.NoError(t, err)
	require.Len(t, flags, 7)
	require.Equal(t, expected, flags)
}

func TestProcessFlagsInvalid(t *testing.T) {
	ctx := &context.Context{}

	source := []string{
		"{{.Version}",
	}

	expected := `template: tmpl:1: unexpected "}" in operand`

	flags, err := processFlags(ctx, &artifact.Artifact{}, []string{}, source, "-testflag=")
	require.EqualError(t, err, expected)
	require.Nil(t, flags)
}

func TestBuildModTimestamp(t *testing.T) {
	// round to seconds since this will be a unix timestamp
	modTime := time.Now().AddDate(-1, 0, 0).Round(1 * time.Second).UTC()

	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)

	config := config.Project{
		Builds: []config.Build{
			{
				ID:     "foo",
				Env:    []string{"GO111MODULE=off"},
				Binary: "bin/foo-{{ .Version }}",
				Targets: []string{
					"linux_amd64",
					"darwin_amd64",
					"windows_amd64",
					"linux_arm_6",
					"js_wasm",
					"linux_mips_softfloat",
					"linux_mips64le_softfloat",
				},
				Asmflags:     []string{".=", "all="},
				Gcflags:      []string{"all="},
				Flags:        []string{"{{.Env.GO_FLAGS}}"},
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				GoBinary:     "go",
			},
		},
	}
	ctx := context.New(config)
	ctx.Env["GO_FLAGS"] = "-v"
	ctx.Git.CurrentTag = "5.6.7"
	ctx.Version = "v" + ctx.Git.CurrentTag
	build := ctx.Config.Builds[0]
	for _, target := range build.Targets {
		var ext string
		if strings.HasPrefix(target, "windows") {
			ext = ".exe"
		} else if target == "js_wasm" {
			ext = ".wasm"
		}
		bin, terr := tmpl.New(ctx).Apply(build.Binary)
		require.NoError(t, terr)

		// injecting some delay here to force inconsistent mod times on bins
		time.Sleep(2 * time.Second)

		err := Default.Build(ctx, build, api.Options{
			Target: target,
			Name:   bin + ext,
			Path:   filepath.Join(folder, "dist", target, bin+ext),
			Ext:    ext,
		})
		require.NoError(t, err)
	}

	for _, bin := range ctx.Artifacts.List() {
		if bin.Type != artifact.Binary {
			continue
		}

		fi, err := os.Stat(bin.Path)
		require.NoError(t, err)
		require.True(t, modTime.Equal(fi.ModTime()), "inconsistent mod times found when specifying ModTimestamp")
	}
}

func TestBuildGoBuildLine(t *testing.T) {
	requireEqualCmd := func(tb testing.TB, build config.Build, expected []string) {
		tb.Helper()
		config := config.Project{
			Builds: []config.Build{build},
		}
		ctx := context.New(config)
		ctx.Version = "v1.2.3"
		ctx.Git.Commit = "aaa"

		line, err := buildGoBuildLine(ctx, config.Builds[0], api.Options{Path: "foo"}, &artifact.Artifact{}, []string{})
		require.NoError(t, err)
		require.Equal(t, expected, line)
	}

	t.Run("full", func(t *testing.T) {
		requireEqualCmd(t, config.Build{
			Main:     ".",
			Asmflags: []string{"asmflag1", "asmflag2"},
			Gcflags:  []string{"gcflag1", "gcflag2"},
			Flags:    []string{"-flag1", "-flag2"},
			Tags:     []string{"tag1", "tag2"},
			Ldflags:  []string{"ldflag1", "ldflag2"},
			GoBinary: "go",
		}, []string{
			"go", "build",
			"-flag1", "-flag2",
			"-asmflags=asmflag1", "-asmflags=asmflag2",
			"-gcflags=gcflag1", "-gcflags=gcflag2",
			"-tags=tag1,tag2",
			"-ldflags=ldflag1 ldflag2",
			"-o", "foo", ".",
		})
	})

	t.Run("simple", func(t *testing.T) {
		requireEqualCmd(t, config.Build{
			Main:     ".",
			GoBinary: "go",
		}, strings.Fields("go build -o foo ."))
	})

	t.Run("ldflags1", func(t *testing.T) {
		requireEqualCmd(t, config.Build{
			Main:     ".",
			Ldflags:  []string{"-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.builtBy=goreleaser"},
			GoBinary: "go",
		}, []string{
			"go", "build",
			"-ldflags=-s -w -X main.version=v1.2.3 -X main.commit=aaa -X main.builtBy=goreleaser",
			"-o", "foo", ".",
		})
	})

	t.Run("ldflags2", func(t *testing.T) {
		requireEqualCmd(t, config.Build{
			Main:     ".",
			Ldflags:  []string{"-s -w", "-X main.version={{.Version}}"},
			GoBinary: "go",
		}, []string{"go", "build", "-ldflags=-s -w -X main.version=v1.2.3", "-o", "foo", "."})
	})
}

//
// Helpers
//

func writeMainWithoutMainFunc(t *testing.T, folder string) {
	t.Helper()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "main.go"),
		[]byte("package main\nconst a = 2\nfunc notMain() {println(0)}"),
		0o644,
	))
}

func writeGoodMain(t *testing.T, folder string) {
	t.Helper()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "main.go"),
		[]byte("package main\nvar a = 1\nfunc main() {println(0)}"),
		0o644,
	))
}

func assertContainsError(t *testing.T, err error, s string) {
	t.Helper()
	require.Error(t, err)
	require.Contains(t, err.Error(), s)
}
