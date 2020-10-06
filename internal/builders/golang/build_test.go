package golang

import (
	"fmt"
	"io/ioutil"
	"os"
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
				"darwin_amd64",
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
	} {
		t.Run(name, func(tt *testing.T) {
			var config = config.Project{
				Builds: []config.Build{
					testcase.build,
				},
			}
			var ctx = context.New(config)
			ctx.Git.CurrentTag = "5.6.7"
			var build = Default.WithDefaults(ctx.Config.Builds[0])
			require.ElementsMatch(t, build.Targets, testcase.targets)
			require.EqualValues(t, testcase.goBinary, build.GoBinary)
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
				GoBinary: "go",
			},
		},
	}
	var ctx = context.New(config)
	ctx.Env["GO_FLAGS"] = "-v"
	ctx.Git.CurrentTag = "5.6.7"
	ctx.Version = "v" + ctx.Git.CurrentTag
	var build = ctx.Config.Builds[0]
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

		var err = Default.Build(ctx, build, api.Options{
			Target: target,
			Name:   bin + ext,
			Path:   filepath.Join(folder, "dist", target, bin+ext),
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
	folder, back := testlib.Mktmp(t)
	defer back()
	subdir := filepath.Join(folder, "bar")
	err := os.Mkdir(subdir, 0755)
	require.NoError(t, err)
	writeGoodMain(t, subdir)
	var config = config.Project{
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
	var ctx = context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	var build = ctx.Config.Builds[0]
	err = Default.Build(ctx, build, api.Options{
		Target: runtimeTarget,
		Name:   build.Binary,
		Path:   filepath.Join(folder, "dist", runtimeTarget, build.Binary),
		Ext:    "",
	})
	require.NoError(t, err)
}

func TestBuildFailed(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)
	var config = config.Project{
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
	var ctx = context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	var err = Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: "darwin_amd64",
	})
	assertContainsError(t, err, `flag provided but not defined: -flag-that-dont-exists-to-force-failure`)
	require.Empty(t, ctx.Artifacts.List())
}

func TestBuildInvalidTarget(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)
	var target = "linux"
	var config = config.Project{
		Builds: []config.Build{
			{
				ID:      "foo",
				Binary:  "foo",
				Targets: []string{target},
			},
		},
	}
	var ctx = context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	var build = ctx.Config.Builds[0]
	var err = Default.Build(ctx, build, api.Options{
		Target: target,
		Name:   build.Binary,
		Path:   filepath.Join(folder, "dist", target, build.Binary),
	})
	require.EqualError(t, err, "linux is not a valid build target")
	require.Len(t, ctx.Artifacts.List(), 0)
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
	ctx.Git.CurrentTag = "5.6.7"
	var err = Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: runtimeTarget,
	})
	require.EqualError(t, err, `template: tmpl:1: unexpected "}" in operand`)
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
	ctx.Git.CurrentTag = "5.6.7"
	var err = Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: runtimeTarget,
	})
	require.EqualError(t, err, `template: tmpl:1: unexpected "}" in operand`)
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
	ctx.Git.CurrentTag = "5.6.7"
	var err = Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: runtimeTarget,
	})
	require.EqualError(t, err, `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunInvalidFlags(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)
	var config = config.Project{
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
	var ctx = context.New(config)
	var err = Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: runtimeTarget,
	})
	require.EqualError(t, err, `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunPipeWithoutMainFunc(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	writeMainWithoutMainFunc(t, folder)
	var config = config.Project{
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
	var ctx = context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	t.Run("empty", func(t *testing.T) {
		ctx.Config.Builds[0].Main = ""
		require.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}), `build for no-main does not contain a main function`)
	})
	t.Run("not main.go", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "foo.go"
		require.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}), `stat foo.go: no such file or directory`)
	})
	t.Run("glob", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "."
		require.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}), `build for no-main does not contain a main function`)
	})
	t.Run("fixed main.go", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "main.go"
		require.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}), `build for no-main does not contain a main function`)
	})
}

func TestRunPipeWithMainFuncNotInMainGoFile(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	require.NoError(t, ioutil.WriteFile(
		filepath.Join(folder, "foo.go"),
		[]byte("package main\nfunc main() {println(0)}"),
		0644,
	))
	var config = config.Project{
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
	var ctx = context.New(config)
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

	var ctx = &context.Context{
		Git: context.GitInfo{
			CurrentTag: "v1.2.3",
			Commit:     "123",
			CommitDate: commit,
		},
		Date:    run,
		Version: "1.2.3",
		Env:     map[string]string{"FOO": "123"},
	}
	var artifact = &artifact.Artifact{Goarch: "amd64"}
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
		t.Run(template, func(tt *testing.T) {
			var ctx = context.New(config.Project{})
			ctx.Git.CurrentTag = "3.4.1"
			flags, err := tmpl.New(ctx).Apply(template)
			require.EqualError(tt, err, eerr)
			require.Empty(tt, flags)
		})
	}
}

func TestProcessFlags(t *testing.T) {
	var ctx = &context.Context{
		Version: "1.2.3",
	}
	ctx.Git.CurrentTag = "5.6.7"

	var artifact = &artifact.Artifact{
		Name:   "name",
		Goos:   "darwin",
		Goarch: "amd64",
		Goarm:  "7",
		Extra: map[string]interface{}{
			"Binary": "binary",
		},
	}

	var source = []string{
		"flag",
		"{{.Version}}",
		"{{.Os}}",
		"{{.Arch}}",
		"{{.Arm}}",
		"{{.Binary}}",
		"{{.ArtifactName}}",
	}

	var expected = []string{
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
	var ctx = &context.Context{}

	var source = []string{
		"{{.Version}",
	}

	var expected = `template: tmpl:1: unexpected "}" in operand`

	flags, err := processFlags(ctx, &artifact.Artifact{}, []string{}, source, "-testflag=")
	require.EqualError(t, err, expected)
	require.Nil(t, flags)
}

func TestJoinLdFlags(t *testing.T) {
	tests := []struct {
		input  []string
		output string
	}{
		{[]string{"-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}} -X main.builtBy=goreleaser"}, "-ldflags=-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}} -X main.builtBy=goreleaser"},
		{[]string{"-s -w", "-X main.version={{.Version}}"}, "-ldflags=-s -w -X main.version={{.Version}}"},
	}

	for _, test := range tests {
		joinedLdFlags := joinLdFlags(test.input)
		require.Equal(t, joinedLdFlags, test.output)
	}
}

func TestBuildModTimestamp(t *testing.T) {
	// round to seconds since this will be a unix timestamp
	modTime := time.Now().AddDate(-1, 0, 0).Round(1 * time.Second).UTC()

	folder, back := testlib.Mktmp(t)
	defer back()
	writeGoodMain(t, folder)

	var config = config.Project{
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
	var ctx = context.New(config)
	ctx.Env["GO_FLAGS"] = "-v"
	ctx.Git.CurrentTag = "5.6.7"
	ctx.Version = "v" + ctx.Git.CurrentTag
	var build = ctx.Config.Builds[0]
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

		var err = Default.Build(ctx, build, api.Options{
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

//
// Helpers
//

func writeMainWithoutMainFunc(t *testing.T, folder string) {
	require.NoError(t, ioutil.WriteFile(
		filepath.Join(folder, "main.go"),
		[]byte("package main\nconst a = 2\nfunc notMain() {println(0)}"),
		0644,
	))
}

func writeGoodMain(t *testing.T, folder string) {
	require.NoError(t, ioutil.WriteFile(
		filepath.Join(folder, "main.go"),
		[]byte("package main\nvar a = 1\nfunc main() {println(0)}"),
		0644,
	))
}

func assertContainsError(t *testing.T, err error, s string) {
	require.Error(t, err)
	require.Contains(t, err.Error(), s)
}
