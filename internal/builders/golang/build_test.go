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

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/experimental"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	api "github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

var runtimeTarget = runtime.GOOS + "_" + runtime.GOARCH

var go118FirstClassAdjustedTargets = []string{
	"darwin_amd64_v1",
	"darwin_arm64_v8.0",
	"linux_386_sse2",
	"linux_amd64_v1",
	"linux_arm_6",
	"linux_arm64_v8.0",
	"windows_386_sse2",
	"windows_amd64_v1",
}

func TestDependencies(t *testing.T) {
	require.NotEmpty(t, Default.Dependencies())
}

func TestParse(t *testing.T) {
	for target, dst := range map[string]Target{
		"linux_amd64": {
			Target:  "linux_amd64_v1",
			Goos:    "linux",
			Goarch:  "amd64",
			Goamd64: "v1",
		},
		"linux_amd64_v2": {
			Target:  "linux_amd64_v2",
			Goos:    "linux",
			Goarch:  "amd64",
			Goamd64: "v2",
		},
		"linux_arm": {
			Target: "linux_arm_" + experimental.DefaultGOARM(),
			Goos:   "linux",
			Goarch: "arm",
			Goarm:  experimental.DefaultGOARM(),
		},
		"linux_arm_7": {
			Target: "linux_arm_7",
			Goos:   "linux",
			Goarch: "arm",
			Goarm:  "7",
		},
		"linux_mips": {
			Target: "linux_mips_hardfloat",
			Goos:   "linux",
			Goarch: "mips",
			Gomips: "hardfloat",
		},
		"linux_mips_softfloat": {
			Target: "linux_mips_softfloat",
			Goos:   "linux",
			Goarch: "mips",
			Gomips: "softfloat",
		},
		"linux_386": {
			Target: "linux_386_sse2",
			Goos:   "linux",
			Goarch: "386",
			Go386:  "sse2",
		},
		"linux_386_hardfloat": {
			Target: "linux_386_hardfloat",
			Goos:   "linux",
			Goarch: "386",
			Go386:  "hardfloat",
		},
		"linux_arm64": {
			Target:  "linux_arm64_v8.0",
			Goos:    "linux",
			Goarch:  "arm64",
			Goarm64: "v8.0",
		},
		"linux_arm64_v9.0": {
			Target:  "linux_arm64_v9.0",
			Goos:    "linux",
			Goarch:  "arm64",
			Goarm64: "v9.0",
		},
	} {
		t.Run(target, func(t *testing.T) {
			got, err := Default.Parse(target)
			require.NoError(t, err)
			require.IsType(t, Target{}, got)
			require.Equal(t, dst, got.(Target))
		})
	}
}

func TestWithDefaults(t *testing.T) {
	for name, testcase := range map[string]struct {
		build   config.Build
		targets []string
		tool    string
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
				Goamd64: []string{
					"v2",
					"v3",
				},
				Tool: "go1.2.3",
			},
			targets: []string{
				"linux_amd64_v2",
				"linux_amd64_v3",
				"linux_mips_softfloat",
				"darwin_amd64_v2",
				"darwin_amd64_v3",
				"windows_amd64_v3",
				"windows_amd64_v2",
				"windows_arm_6",
				"linux_arm_6",
			},
			tool: "go1.2.3",
		},
		"empty": {
			build: config.Build{
				ID:     "foo2",
				Binary: "foo",
			},
			targets: []string{
				"linux_amd64_v1",
				"linux_386_sse2",
				"linux_arm64_v8.0",
				"darwin_amd64_v1",
				"darwin_arm64_v8.0",
				"windows_amd64_v1",
				"windows_arm64_v8.0",
				"windows_386_sse2",
			},
			tool: "go",
		},
		"custom targets": {
			build: config.Build{
				ID:     "foo3",
				Binary: "foo",
				Targets: []string{
					"linux_386_sse2",
					"darwin_amd64_v2",
				},
			},
			targets: []string{
				"linux_386_sse2",
				"darwin_amd64_v2",
			},
			tool: "go",
		},
		"custom targets no amd64": {
			build: config.Build{
				ID:     "foo3",
				Binary: "foo",
				Targets: []string{
					"linux_386_sse2",
					"darwin_amd64",
				},
			},
			targets: []string{
				"linux_386_sse2",
				"darwin_amd64_v1",
			},
			tool: "go",
		},
		"custom targets no arm": {
			build: config.Build{
				ID:      "foo3",
				Binary:  "foo",
				Targets: []string{"linux_arm"},
			},
			targets: []string{"linux_arm_6"},
			tool:    "go",
		},
		"custom targets no arm64": {
			build: config.Build{
				ID:      "foo3",
				Binary:  "foo",
				Targets: []string{"linux_arm64"},
			},
			targets: []string{"linux_arm64_v8.0"},
			tool:    "go",
		},
		"custom targets no ppc64": {
			build: config.Build{
				ID:      "foo3",
				Binary:  "foo",
				Targets: []string{"linux_ppc64le", "linux_ppc64"},
			},
			targets: []string{"linux_ppc64le_power8", "linux_ppc64_power8"},
			tool:    "go",
		},
		"custom targets no riscv64": {
			build: config.Build{
				ID:      "foo3",
				Binary:  "foo",
				Targets: []string{"linux_riscv64"},
			},
			targets: []string{"linux_riscv64_rva20u64"},
			tool:    "go",
		},
		"custom targets no mips": {
			build: config.Build{
				ID:      "foo3",
				Binary:  "foo",
				Targets: []string{"linux_mips"},
			},
			targets: []string{"linux_mips_hardfloat"},
			tool:    "go",
		},
		"custom targets no mipsle": {
			build: config.Build{
				ID:      "foo3",
				Binary:  "foo",
				Targets: []string{"linux_mipsle"},
			},
			targets: []string{"linux_mipsle_hardfloat"},
			tool:    "go",
		},
		"custom targets no mips64": {
			build: config.Build{
				ID:      "foo3",
				Binary:  "foo",
				Targets: []string{"linux_mips64"},
			},
			targets: []string{"linux_mips64_hardfloat"},
			tool:    "go",
		},
		"custom targets no mips64le": {
			build: config.Build{
				ID:      "foo3",
				Binary:  "foo",
				Targets: []string{"linux_mips64le"},
			},
			targets: []string{"linux_mips64le_hardfloat"},
			tool:    "go",
		},
		"empty with custom dir": {
			build: config.Build{
				ID:     "foo2",
				Binary: "foo",
				Dir:    "./testdata",
			},
			targets: []string{
				"linux_amd64_v1",
				"linux_386_sse2",
				"linux_arm64_v8.0",
				"darwin_amd64_v1",
				"darwin_arm64_v8.0",
				"windows_amd64_v1",
				"windows_arm64_v8.0",
				"windows_386_sse2",
			},
			tool: "go",
		},
		"empty with custom dir that doesn't exist": {
			build: config.Build{
				ID:     "foo2",
				Binary: "foo",
				Dir:    "./nope",
			},
			targets: []string{
				"linux_amd64_v1",
				"linux_386_sse2",
				"linux_arm64_v8.0",
				"darwin_amd64_v1",
				"darwin_arm64_v8.0",
				"windows_amd64_v1",
				"windows_arm64_v8.0",
				"windows_386_sse2",
			},
			tool: "go",
		},
		"go first class targets": {
			build: config.Build{
				ID:      "foo3",
				Binary:  "foo",
				Targets: []string{goStableFirstClassTargetsName},
			},
			targets: go118FirstClassAdjustedTargets,
			tool:    "go",
		},
		"go 1.18 first class targets": {
			build: config.Build{
				ID:      "foo3",
				Binary:  "foo",
				Targets: []string{go118FirstClassTargetsName},
			},
			targets: go118FirstClassAdjustedTargets,
			tool:    "go",
		},
		"go 1.18 first class targets plus custom": {
			build: config.Build{
				ID:      "foo3",
				Binary:  "foo",
				Targets: []string{"linux_amd64_v1", go118FirstClassTargetsName, "darwin_amd64_v2"},
			},
			targets: append(go118FirstClassAdjustedTargets, "darwin_amd64_v2"),
			tool:    "go",
		},
		"repeating targets": {
			build: config.Build{
				ID:      "foo3",
				Binary:  "foo",
				Targets: []string{go118FirstClassTargetsName, go118FirstClassTargetsName, goStableFirstClassTargetsName},
			},
			targets: go118FirstClassAdjustedTargets,
			tool:    "go",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if testcase.build.Tool != "" && testcase.build.Tool != "go" {
				createFakeGoBinaryWithVersion(t, testcase.build.Tool, "go1.18")
			}
			ctx := testctx.NewWithCfg(config.Project{
				Builds: []config.Build{
					testcase.build,
				},
			}, testctx.WithCurrentTag("5.6.7"))
			build, err := Default.WithDefaults(ctx.Config.Builds[0])
			require.NoError(t, err)
			require.ElementsMatch(t, build.Targets, testcase.targets)
			require.Equal(t, testcase.tool, build.Tool)
		})
	}
}

func TestDefaults(t *testing.T) {
	t.Run("command not set", func(t *testing.T) {
		build, err := Default.WithDefaults(config.Build{})
		require.NoError(t, err)
		require.Equal(t, "build", build.Command)
	})
	t.Run("command set", func(t *testing.T) {
		build, err := Default.WithDefaults(config.Build{
			Command: "test",
		})
		require.NoError(t, err)
		require.Equal(t, "test", build.Command)
	})
}

// createFakeGoBinaryWithVersion creates a temporary executable with the
// given name, which will output a go version string with the given version.
//
// The temporary directory created by this function will be placed in the
// PATH variable for the duration of (and cleaned up at the end of) the
// current test run.
func createFakeGoBinaryWithVersion(tb testing.TB, name, version string) {
	tb.Helper()
	d := tb.TempDir()

	require.NoError(tb, os.WriteFile(
		filepath.Join(d, name),
		fmt.Appendf(nil, "#!/bin/sh\necho %s", version),
		0o755,
	))

	currentPath := os.Getenv("PATH")

	path := fmt.Sprintf("%s%c%s", d, os.PathListSeparator, currentPath)
	tb.Setenv("PATH", path)
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
		"goamd64": {
			build: config.Build{
				Goarch:  []string{"amd64"},
				Goamd64: []string{"v1", "v431"},
			},
			expectedErr: "invalid goamd64: v431",
		},
	} {
		t.Run(s, func(t *testing.T) {
			ctx := testctx.NewWithCfg(config.Project{
				Builds: []config.Build{
					tc.build,
				},
			})
			_, err := Default.WithDefaults(ctx.Config.Builds[0])
			require.EqualError(t, err, tc.expectedErr)
		})
	}
}

func TestBuild(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	ctx := testctx.NewWithCfg(config.Project{
		Env: []string{"GO_FLAGS=-v", "GOBIN=go"},
		Builds: []config.Build{
			{
				ID:     "foo",
				Binary: "bin/foo-{{ .Version }}",
				Targets: []string{
					"linux_amd64",
					"darwin_amd64",
					"windows_amd64",
					"linux_arm_6",
					"js_wasm",
					"linux_mips_softfloat",
				},
				Tool:    "{{ .Env.GOBIN }}",
				Command: "build",
				BuildDetailsOverrides: []config.BuildDetailsOverride{
					{
						Goos:   "linux",
						Goarch: "amd64",
						BuildDetails: config.BuildDetails{
							Env: []string{"TEST_O=1"},
						},
					},
				},
				BuildDetails: config.BuildDetails{
					Env: []string{
						"GO111MODULE=off",
						`TEST_T={{- if eq .Os "windows" -}}
						w
						{{- else if eq .Os "darwin" -}}
						d
						{{- else if eq .Os "linux" -}}
						l
						{{- end -}}`,
					},
					Asmflags: []string{".=", "all="},
					Gcflags:  []string{"all="},
					Flags:    []string{"{{.Env.GO_FLAGS}}"},
					Tags:     []string{"osusergo", "netgo", "static_build"},
				},
			},
		},
	}, testctx.WithCurrentTag("v5.6.7"), testctx.WithVersion("v5.6.7"))
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

		gtarget, err := Default.Parse(target)
		require.NoError(t, err)
		require.NoError(t, Default.Build(ctx, build, api.Options{
			Target: gtarget,
			Name:   bin + ext,
			Path:   filepath.Join(folder, "dist", target, bin+ext),
			Ext:    ext,
		}))
	}
	list := ctx.Artifacts
	require.NoError(t, list.Visit(func(a *artifact.Artifact) error {
		s, err := filepath.Rel(folder, a.Path)
		if err == nil {
			a.Path = s
		}
		return nil
	}))
	expected := []*artifact.Artifact{
		{
			Name:    "bin/foo-v5.6.7",
			Path:    filepath.ToSlash(filepath.Join("dist", "linux_amd64", "bin", "foo-v5.6.7")),
			Goos:    "linux",
			Goarch:  "amd64",
			Goamd64: "v1",
			Target:  "linux_amd64_v1",
			Type:    artifact.Binary,
			Extra: map[string]any{
				artifact.ExtraExt:     "",
				artifact.ExtraBinary:  "foo-v5.6.7",
				artifact.ExtraID:      "foo",
				artifact.ExtraBuilder: "go",
				"testEnvs":            []string{"TEST_T=l", "TEST_O=1"},
			},
		},
		{
			Name:   "bin/foo-v5.6.7",
			Path:   filepath.ToSlash(filepath.Join("dist", "linux_mips_softfloat", "bin", "foo-v5.6.7")),
			Goos:   "linux",
			Goarch: "mips",
			Gomips: "softfloat",
			Target: "linux_mips_softfloat",
			Type:   artifact.Binary,
			Extra: map[string]any{
				artifact.ExtraExt:     "",
				artifact.ExtraBinary:  "foo-v5.6.7",
				artifact.ExtraID:      "foo",
				artifact.ExtraBuilder: "go",
				"testEnvs":            []string{"TEST_T=l"},
			},
		},
		{
			Name:    "bin/foo-v5.6.7",
			Path:    filepath.ToSlash(filepath.Join("dist", "darwin_amd64", "bin", "foo-v5.6.7")),
			Goos:    "darwin",
			Goarch:  "amd64",
			Goamd64: "v1",
			Target:  "darwin_amd64_v1",
			Type:    artifact.Binary,
			Extra: map[string]any{
				artifact.ExtraExt:     "",
				artifact.ExtraBinary:  "foo-v5.6.7",
				artifact.ExtraID:      "foo",
				artifact.ExtraBuilder: "go",
				"testEnvs":            []string{"TEST_T=d"},
			},
		},
		{
			Name:   "bin/foo-v5.6.7",
			Path:   filepath.ToSlash(filepath.Join("dist", "linux_arm_6", "bin", "foo-v5.6.7")),
			Goos:   "linux",
			Goarch: "arm",
			Goarm:  "6",
			Target: "linux_arm_6",
			Type:   artifact.Binary,
			Extra: map[string]any{
				artifact.ExtraExt:     "",
				artifact.ExtraBinary:  "foo-v5.6.7",
				artifact.ExtraID:      "foo",
				artifact.ExtraBuilder: "go",
				"testEnvs":            []string{"TEST_T=l"},
			},
		},
		{
			Name:    "bin/foo-v5.6.7.exe",
			Path:    filepath.ToSlash(filepath.Join("dist", "windows_amd64", "bin", "foo-v5.6.7.exe")),
			Goos:    "windows",
			Goarch:  "amd64",
			Goamd64: "v1",
			Target:  "windows_amd64_v1",
			Type:    artifact.Binary,
			Extra: map[string]any{
				artifact.ExtraExt:     ".exe",
				artifact.ExtraBinary:  "foo-v5.6.7",
				artifact.ExtraID:      "foo",
				artifact.ExtraBuilder: "go",
				"testEnvs":            []string{"TEST_T=w"},
			},
		},
		{
			Name:   "bin/foo-v5.6.7.wasm",
			Path:   filepath.ToSlash(filepath.Join("dist", "js_wasm", "bin", "foo-v5.6.7.wasm")),
			Goos:   "js",
			Goarch: "wasm",
			Target: "js_wasm",
			Type:   artifact.Binary,
			Extra: map[string]any{
				artifact.ExtraExt:     ".wasm",
				artifact.ExtraBinary:  "foo-v5.6.7",
				artifact.ExtraID:      "foo",
				artifact.ExtraBuilder: "go",
				"testEnvs":            []string{"TEST_T="},
			},
		},
	}

	got := list.List()
	testlib.RequireEqualArtifacts(t, expected, got)
}

func TestBuildInvalidEnv(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	ctx := testctx.NewWithCfg(config.Project{
		Builds: []config.Build{
			{
				ID:     "foo",
				Dir:    ".",
				Binary: "foo",
				Targets: []string{
					runtimeTarget,
				},
				Tool: "go",
				BuildDetails: config.BuildDetails{
					Env: []string{"GO111MODULE={{ .Nope }}"},
				},
			},
		},
	}, testctx.WithCurrentTag("5.6.7"))
	build := ctx.Config.Builds[0]
	err := Default.Build(ctx, build, api.Options{
		Target: mustParse(t, runtimeTarget),
		Name:   build.Binary,
		Path:   filepath.Join("dist", runtimeTarget, build.Binary),
		Ext:    "",
	})
	testlib.RequireTemplateError(t, err)
}

func TestBuildCodeInSubdir(t *testing.T) {
	folder := testlib.Mktmp(t)
	subdir := filepath.Join(folder, "bar")
	err := os.Mkdir(subdir, 0o755)
	require.NoError(t, err)
	writeGoodMain(t, subdir)
	ctx := testctx.NewWithCfg(config.Project{
		Builds: []config.Build{
			{
				ID:     "foo",
				Dir:    "bar",
				Binary: "foo",
				Targets: []string{
					runtimeTarget,
				},
				Tool:    "go",
				Command: "build",
				BuildDetails: config.BuildDetails{
					Env: []string{"GO111MODULE=off"},
				},
			},
		},
	}, testctx.WithCurrentTag("5.6.7"))
	build := ctx.Config.Builds[0]
	err = Default.Build(ctx, build, api.Options{
		Target: mustParse(t, runtimeTarget),
		Name:   build.Binary,
		Path:   filepath.Join("dist", runtimeTarget, build.Binary),
		Ext:    "",
	})
	require.NoError(t, err)
}

func TestBuildWithDotGoDir(t *testing.T) {
	folder := testlib.Mktmp(t)
	require.NoError(t, os.Mkdir(filepath.Join(folder, ".go"), 0o755))
	writeGoodMain(t, folder)
	ctx := testctx.NewWithCfg(config.Project{
		Builds: []config.Build{
			{
				ID:      "foo",
				Binary:  "foo",
				Targets: []string{runtimeTarget},
				Tool:    "go",
				Command: "build",
				BuildDetails: config.BuildDetails{
					Env: []string{"GO111MODULE=off"},
				},
			},
		},
	}, testctx.WithCurrentTag("5.6.7"))
	build := ctx.Config.Builds[0]
	require.NoError(t, Default.Build(ctx, build, api.Options{
		Target: mustParse(t, runtimeTarget),
		Name:   build.Binary,
		Path:   filepath.Join("dist", runtimeTarget, build.Binary),
		Ext:    "",
	}))
}

func TestBuildFailed(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	ctx := testctx.NewWithCfg(config.Project{
		Builds: []config.Build{
			{
				ID: "buildid",
				BuildDetails: config.BuildDetails{
					Flags: []string{"-flag-that-dont-exists-to-force-failure"},
				},
				Targets: []string{
					runtimeTarget,
				},
				Tool:    "go",
				Command: "build",
			},
		},
	}, testctx.WithCurrentTag("5.6.7"))
	err := Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: mustParse(t, "darwin_amd64"),
	})
	require.ErrorContains(t, err, `flag provided but not defined: -flag-that-dont-exists-to-force-failure`)
	require.Empty(t, ctx.Artifacts.List())
}

func TestRunInvalidAsmflags(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	ctx := testctx.NewWithCfg(config.Project{
		Builds: []config.Build{
			{
				Binary: "nametest",
				BuildDetails: config.BuildDetails{
					Asmflags: []string{"{{.Version}"},
				},
				Targets: []string{
					runtimeTarget,
				},
			},
		},
	}, testctx.WithCurrentTag("5.6.7"))
	err := Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: mustParse(t, runtimeTarget),
	})
	testlib.RequireTemplateError(t, err)
}

func TestRunInvalidGcflags(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	ctx := testctx.NewWithCfg(config.Project{
		Builds: []config.Build{
			{
				Binary: "nametest",
				BuildDetails: config.BuildDetails{
					Gcflags: []string{"{{.Version}"},
				},
				Targets: []string{
					runtimeTarget,
				},
			},
		},
	}, testctx.WithCurrentTag("5.6.7"))
	err := Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: mustParse(t, runtimeTarget),
	})
	testlib.RequireTemplateError(t, err)
}

func TestRunInvalidLdflags(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	ctx := testctx.NewWithCfg(config.Project{
		Builds: []config.Build{
			{
				Binary: "nametest",
				BuildDetails: config.BuildDetails{
					Flags:   []string{"-v"},
					Ldflags: []string{"-s -w -X main.version={{.Version}"},
				},
				Targets: []string{
					runtimeTarget,
				},
			},
		},
	}, testctx.WithCurrentTag("5.6.7"))
	err := Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: mustParse(t, runtimeTarget),
	})
	testlib.RequireTemplateError(t, err)
}

func TestRunInvalidFlags(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	ctx := testctx.NewWithCfg(config.Project{
		Builds: []config.Build{
			{
				Binary: "nametest",
				BuildDetails: config.BuildDetails{
					Flags: []string{"{{.Env.GOOS}"},
				},
				Targets: []string{
					runtimeTarget,
				},
			},
		},
	})
	err := Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: mustParse(t, runtimeTarget),
	})
	testlib.RequireTemplateError(t, err)
}

func TestRunPipeWithoutMainFunc(t *testing.T) {
	newCtx := func(t *testing.T) *context.Context {
		t.Helper()
		folder := testlib.Mktmp(t)
		writeMainWithoutMainFunc(t, folder)
		ctx := testctx.NewWithCfg(config.Project{
			Builds: []config.Build{{Binary: "no-main"}},
		}, testctx.WithCurrentTag("5.6.7"))
		return ctx
	}
	t.Run("empty", func(t *testing.T) {
		ctx := newCtx(t)
		ctx.Config.Builds[0].Main = ""
		require.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: mustParse(t, runtimeTarget),
		}), errNoMain{"no-main"}.Error())
	})
	t.Run("not main.go", func(t *testing.T) {
		ctx := newCtx(t)
		ctx.Config.Builds[0].Main = "foo.go"
		require.ErrorIs(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: mustParse(t, runtimeTarget),
		}), os.ErrNotExist)
	})
	t.Run("glob", func(t *testing.T) {
		ctx := newCtx(t)
		ctx.Config.Builds[0].Main = "."
		require.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: mustParse(t, runtimeTarget),
		}), errNoMain{"no-main"}.Error())
	})
	t.Run("fixed main.go", func(t *testing.T) {
		ctx := newCtx(t)
		ctx.Config.Builds[0].Main = "main.go"
		require.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: mustParse(t, runtimeTarget),
		}), errNoMain{"no-main"}.Error())
	})
	t.Run("using gomod.proxy", func(t *testing.T) {
		ctx := newCtx(t)
		ctx.Config.GoMod.Proxy = true
		ctx.Config.Builds[0].Dir = "dist/proxy/test"
		ctx.Config.Builds[0].Main = "github.com/caarlos0/test"
		ctx.Config.Builds[0].UnproxiedDir = "."
		ctx.Config.Builds[0].UnproxiedMain = "."
		require.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: mustParse(t, runtimeTarget),
		}), errNoMain{"no-main"}.Error())
	})
}

func TestBuildTests(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeTest(t, folder)
	ctx := testctx.NewWithCfg(config.Project{
		Builds: []config.Build{{
			Binary:  "foo.test",
			Command: "test",
			BuildDetails: config.BuildDetails{
				Flags: []string{"-c"},
			},
			NoMainCheck: true,
		}},
	}, testctx.WithCurrentTag("5.6.7"))
	build, err := Default.WithDefaults(ctx.Config.Builds[0])
	require.NoError(t, err)
	require.NoError(t, Default.Build(ctx, build, api.Options{
		Target: mustParse(t, runtimeTarget),
	}))
}

func TestRunPipeWithProxiedRepo(t *testing.T) {
	folder := testlib.Mktmp(t)
	out, err := exec.CommandContext(t.Context(), "git", "clone", "https://github.com/goreleaser/test-mod", "-b", "v0.1.1", "--depth=1", ".").CombinedOutput()
	require.NoError(t, err, string(out))

	proxied := filepath.Join(folder, "dist/proxy/default")
	require.NoError(t, os.MkdirAll(proxied, 0o750))
	require.NoError(t, os.WriteFile(
		filepath.Join(proxied, "main.go"),
		[]byte(`// +build main
package main

import _ "github.com/goreleaser/test-mod"
`),
		0o666,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(proxied, "go.mod"),
		[]byte("module foo\nrequire github.com/goreleaser/test-mod v0.1.1"),
		0o666,
	))

	cmd := exec.CommandContext(t.Context(), "go", "mod", "tidy")
	cmd.Dir = proxied
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	ctx := testctx.NewWithCfg(config.Project{
		GoMod: config.GoMod{
			Proxy: true,
		},
		Builds: []config.Build{
			{
				Binary:        "foo",
				Main:          "github.com/goreleaser/test-mod",
				Dir:           proxied,
				UnproxiedMain: ".",
				UnproxiedDir:  ".",
				Targets: []string{
					runtimeTarget,
				},
				Tool:    "go",
				Command: "build",
			},
		},
	})

	require.NoError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: mustParse(t, runtimeTarget),
	}))
}

func TestRunPipeWithMainFuncNotInMainGoFile(t *testing.T) {
	folder := testlib.Mktmp(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "foo.go"),
		[]byte("package main\nfunc main() {println(0)}"),
		0o644,
	))
	ctx := testctx.NewWithCfg(config.Project{
		Builds: []config.Build{
			{
				Binary: "foo",
				Hooks:  config.BuildHookConfig{},
				Targets: []string{
					runtimeTarget,
				},
				BuildDetails: config.BuildDetails{
					Env: []string{"GO111MODULE=off"},
				},
				Tool:    "go",
				Command: "build",
			},
		},
	}, testctx.WithCurrentTag("5.6.7"))
	t.Run("empty", func(t *testing.T) {
		ctx.Config.Builds[0].Main = ""
		require.NoError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: mustParse(t, runtimeTarget),
		}))
	})
	t.Run("foo.go", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "foo.go"
		require.NoError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: mustParse(t, runtimeTarget),
		}))
	})
	t.Run("glob", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "."
		require.NoError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: mustParse(t, runtimeTarget),
		}))
	})
}

func TestLdFlagsFullTemplate(t *testing.T) {
	run := time.Now().UTC()
	commit := time.Now().AddDate(-1, 0, 0)
	ctx := testctx.New(
		testctx.WithCurrentTag("v1.2.3"),
		testctx.WithCommit("123"),
		testctx.WithCommitDate(commit),
		testctx.WithVersion("1.2.3"),
		testctx.WithEnv(map[string]string{"FOO": "123"}),
		testctx.WithDate(run),
	)
	artifact := &artifact.Artifact{Goarch: "amd64"}
	flags, err := tmpl.New(ctx).WithArtifact(artifact).
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
	for _, template := range []string{
		"{{ .Nope }",
		"{{.Env.NOPE}}",
	} {
		t.Run(template, func(t *testing.T) {
			ctx := testctx.New(testctx.WithCurrentTag("3.4.1"))
			flags, err := tmpl.New(ctx).Apply(template)
			testlib.RequireTemplateError(t, err)
			require.Empty(t, flags)
		})
	}
}

func TestBuildModTimestamp(t *testing.T) {
	// round to seconds since this will be a unix timestamp
	modTime := time.Now().AddDate(-1, 0, 0).Round(time.Second).UTC()

	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)

	ctx := testctx.NewWithCfg(
		config.Project{
			Env: []string{"GO_FLAGS=-v"},
			Builds: []config.Build{{
				ID:     "foo",
				Binary: "bin/foo-{{ .Version }}",
				Targets: []string{
					"linux_amd64",
					"darwin_amd64",
					"linux_arm_6",
					"linux_mips_softfloat",
					"linux_mips64le_softfloat",
				},
				BuildDetails: config.BuildDetails{
					Env:      []string{"GO111MODULE=off"},
					Asmflags: []string{".=", "all="},
					Gcflags:  []string{"all="},
					Flags:    []string{"{{.Env.GO_FLAGS}}"},
				},
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				Tool:         "go",
				Command:      "build",
			}},
		},
		testctx.WithCurrentTag("v5.6.7"),
		testctx.WithVersion("5.6.7"),
	)

	build := ctx.Config.Builds[0]
	for _, target := range build.Targets {
		bin, terr := tmpl.New(ctx).Apply(build.Binary)
		require.NoError(t, terr)

		err := Default.Build(ctx, build, api.Options{
			Target: mustParse(t, runtimeTarget),
			Name:   bin,
			Path:   filepath.Join(folder, "dist", target, bin),
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
		ctx := testctx.NewWithCfg(
			config.Project{
				Builds: []config.Build{build},
			},
			testctx.WithVersion("1.2.3"),
			testctx.WithGitInfo(context.GitInfo{Commit: "aaa"}),
			testctx.WithEnv(map[string]string{"GOBIN": "go"}),
		)
		options := api.Options{
			Path:   ctx.Config.Builds[0].Binary,
			Target: mustParse(t, "linux_amd64"),
		}

		dets, err := withOverrides(ctx, build, options.Target.(Target))
		require.NoError(t, err)

		line, err := buildGoBuildLine(
			ctx,
			build,
			dets,
			options,
			&artifact.Artifact{},
			[]string{},
		)
		require.NoError(t, err)
		require.Equal(t, expected, line)
	}

	t.Run("full", func(t *testing.T) {
		requireEqualCmd(t, config.Build{
			Main: ".",
			BuildDetails: config.BuildDetails{
				Asmflags: []string{"asmflag1", "asmflag2"},
				Gcflags:  []string{"gcflag1", "gcflag2"},
				Flags:    []string{"-flag1", "-flag2"},
				Tags:     []string{"tag1", "tag2"},
				Ldflags:  []string{"ldflag1", "ldflag2"},
			},
			Binary:  "foo",
			Tool:    "{{ .Env.GOBIN }}",
			Command: "build",
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

	t.Run("with overrides", func(t *testing.T) {
		requireEqualCmd(t, config.Build{
			Main: ".",
			BuildDetails: config.BuildDetails{
				Asmflags: []string{"asmflag1", "asmflag2"},
				Gcflags:  []string{"gcflag1", "gcflag2"},
				Flags:    []string{"-flag1", "-flag2"},
				Tags:     []string{"tag1", "tag2"},
				Ldflags:  []string{"ldflag1", "ldflag2"},
			},
			BuildDetailsOverrides: []config.BuildDetailsOverride{
				{
					Goos:   "linux",
					Goarch: "amd64",
					BuildDetails: config.BuildDetails{
						Asmflags: []string{"asmflag3"},
						Gcflags:  []string{"gcflag3"},
						Flags:    []string{"-flag3"},
						Tags:     []string{"tag3"},
						Ldflags:  []string{"ldflag3"},
					},
				},
			},
			Tool:    "go",
			Binary:  "foo",
			Command: "build",
		}, []string{
			"go", "build",
			"-flag3",
			"-asmflags=asmflag3",
			"-gcflags=gcflag3",
			"-tags=tag3",
			"-ldflags=ldflag3",
			"-o", "foo", ".",
		})
	})

	t.Run("simple", func(t *testing.T) {
		requireEqualCmd(t, config.Build{
			Main:    ".",
			Tool:    "go",
			Command: "build",
			Binary:  "foo",
		}, strings.Fields("go build -o foo ."))
	})

	t.Run("test", func(t *testing.T) {
		requireEqualCmd(t, config.Build{
			Main:    ".",
			Tool:    "go",
			Command: "test",
			Binary:  "foo.test",
			BuildDetails: config.BuildDetails{
				Flags: []string{"-c"},
			},
		}, strings.Fields("go test -c -o foo.test ."))
	})

	t.Run("build test always as c flags", func(t *testing.T) {
		requireEqualCmd(t, config.Build{
			Main:    ".",
			Tool:    "go",
			Command: "test",
			Binary:  "foo.test",
		}, strings.Fields("go test -c -o foo.test ."))
	})

	t.Run("ldflags1", func(t *testing.T) {
		requireEqualCmd(t, config.Build{
			Main: ".",
			BuildDetails: config.BuildDetails{
				Ldflags: []string{"-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.builtBy=goreleaser"},
			},
			Tool:    "go",
			Command: "build",
			Binary:  "foo",
		}, []string{
			"go", "build",
			"-ldflags=-s -w -X main.version=1.2.3 -X main.commit=aaa -X main.builtBy=goreleaser",
			"-o", "foo", ".",
		})
	})

	t.Run("ldflags2", func(t *testing.T) {
		requireEqualCmd(t, config.Build{
			Main: ".",
			BuildDetails: config.BuildDetails{
				Ldflags: []string{"-s -w", "-X main.version={{.Version}}"},
			},
			Tool:    "go",
			Binary:  "foo",
			Command: "build",
		}, []string{"go", "build", "-ldflags=-s -w -X main.version=1.2.3", "-o", "foo", "."})
	})
}

func TestOverrides(t *testing.T) {
	for _, arch := range []string{
		"amd64",
		"arm64",
		"ppc64",
		"ppc64le",
		"riscv64",
		"386",
		"mips",
	} {
		t.Run("linux "+arch, func(t *testing.T) {
			dets, err := withOverrides(
				testctx.New(),
				config.Build{
					BuildDetails: config.BuildDetails{
						Ldflags: []string{"original"},
						Env:     []string{"BAR=foo", "FOO=bar"},
					},
					BuildDetailsOverrides: []config.BuildDetailsOverride{
						{
							Goos:   "linux",
							Goarch: arch,
							BuildDetails: config.BuildDetails{
								Ldflags: []string{"overridden"},
								Env:     []string{"FOO=overridden"},
							},
						},
					},
				}, mustParse(t, "linux_"+arch),
			)
			require.NoError(t, err)
			require.ElementsMatch(t, dets.Ldflags, []string{"overridden"})
			require.ElementsMatch(t, dets.Env, []string{"BAR=foo", "FOO=overridden"})
		})
	}

	t.Run("single sided", func(t *testing.T) {
		dets, err := withOverrides(
			testctx.New(),
			config.Build{
				BuildDetails: config.BuildDetails{},
				BuildDetailsOverrides: []config.BuildDetailsOverride{
					{
						Goos:   "linux",
						Goarch: "amd64",
						BuildDetails: config.BuildDetails{
							Ldflags:  []string{"overridden"},
							Tags:     []string{"tag1"},
							Asmflags: []string{"asm1"},
							Gcflags:  []string{"gcflag1"},
						},
					},
				},
			}, mustParse(t, "linux_amd64"),
		)
		require.NoError(t, err)
		require.Equal(t, config.BuildDetails{
			Ldflags:  []string{"overridden"},
			Gcflags:  []string{"gcflag1"},
			Asmflags: []string{"asm1"},
			Tags:     []string{"tag1"},
			Env:      []string{},
		}, dets)
	})

	t.Run("with template", func(t *testing.T) {
		dets, err := withOverrides(
			testctx.New(),
			config.Build{
				BuildDetails: config.BuildDetails{
					Ldflags:  []string{"original"},
					Asmflags: []string{"asm1"},
				},
				BuildDetailsOverrides: []config.BuildDetailsOverride{
					{
						Goos:   "{{ .Runtime.Goos }}",
						Goarch: "{{ .Runtime.Goarch }}",
						BuildDetails: config.BuildDetails{
							Ldflags: []string{"overridden"},
						},
					},
				},
			}, mustParse(t, runtimeTarget),
		)
		require.NoError(t, err)
		require.Equal(t, config.BuildDetails{
			Ldflags:  []string{"overridden"},
			Asmflags: []string{"asm1"},
			Env:      []string{},
		}, dets)
	})

	t.Run("with invalid template", func(t *testing.T) {
		_, err := withOverrides(
			testctx.New(),
			config.Build{
				BuildDetailsOverrides: []config.BuildDetailsOverride{
					{
						Goos: "{{ .Runtime.Goos }",
					},
				},
			}, mustParse(t, runtimeTarget),
		)
		testlib.RequireTemplateError(t, err)
	})

	t.Run("with goarm64", func(t *testing.T) {
		dets, err := withOverrides(
			testctx.New(),
			config.Build{
				BuildDetails: config.BuildDetails{
					Ldflags: []string{"original"},
				},
				BuildDetailsOverrides: []config.BuildDetailsOverride{
					{
						Goos:    "linux",
						Goarch:  "arm64",
						Goarm64: "v8.0",
						BuildDetails: config.BuildDetails{
							Ldflags: []string{"overridden"},
						},
					},
				},
			}, mustParse(t, "linux_arm64_v8.0"),
		)
		require.NoError(t, err)
		require.Equal(t, config.BuildDetails{
			Ldflags: []string{"overridden"},
			Env:     []string{},
		}, dets)
	})

	t.Run("with goarm64 unspecified", func(t *testing.T) {
		dets, err := withOverrides(
			testctx.New(),
			config.Build{
				BuildDetails: config.BuildDetails{
					Ldflags: []string{"original"},
				},
				BuildDetailsOverrides: []config.BuildDetailsOverride{
					{
						Goos:   "linux",
						Goarch: "arm64",
						BuildDetails: config.BuildDetails{
							Ldflags: []string{"overridden"},
						},
					},
				},
			}, mustParse(t, "linux_arm64_v8.0"),
		)
		require.NoError(t, err)
		require.Equal(t, config.BuildDetails{
			Ldflags: []string{"overridden"},
			Env:     []string{},
		}, dets)
	})

	t.Run("with goarm", func(t *testing.T) {
		dets, err := withOverrides(
			testctx.New(),
			config.Build{
				BuildDetails: config.BuildDetails{
					Ldflags: []string{"original"},
				},
				BuildDetailsOverrides: []config.BuildDetailsOverride{
					{
						Goos:   "linux",
						Goarch: "arm",
						Goarm:  "6",
						BuildDetails: config.BuildDetails{
							Ldflags: []string{"overridden"},
						},
					},
				},
			}, mustParse(t, "linux_arm_6"),
		)
		require.NoError(t, err)
		require.Equal(t, config.BuildDetails{
			Ldflags: []string{"overridden"},
			Env:     []string{},
		}, dets)
	})

	t.Run("with goarm unspecified", func(t *testing.T) {
		dets, err := withOverrides(
			testctx.New(),
			config.Build{
				BuildDetails: config.BuildDetails{
					Ldflags: []string{"original"},
				},
				BuildDetailsOverrides: []config.BuildDetailsOverride{
					{
						Goos:   "linux",
						Goarch: "arm",
						BuildDetails: config.BuildDetails{
							Ldflags: []string{"overridden"},
						},
					},
				},
			}, mustParse(t, "linux_arm_6"),
		)
		require.NoError(t, err)
		require.Equal(t, config.BuildDetails{
			Ldflags: []string{"overridden"},
			Env:     []string{},
		}, dets)
	})

	t.Run("with gomips", func(t *testing.T) {
		dets, err := withOverrides(
			testctx.New(),
			config.Build{
				BuildDetails: config.BuildDetails{
					Ldflags: []string{"original"},
				},
				BuildDetailsOverrides: []config.BuildDetailsOverride{
					{
						Goos:   "linux",
						Goarch: "mips",
						Gomips: "softfloat",
						BuildDetails: config.BuildDetails{
							Ldflags: []string{"overridden"},
						},
					},
				},
			}, mustParse(t, "linux_mips_softfloat"),
		)
		require.NoError(t, err)
		require.Equal(t, config.BuildDetails{
			Ldflags: []string{"overridden"},
			Env:     []string{},
		}, dets)
	})

	t.Run("with gomips unspecified", func(t *testing.T) {
		dets, err := withOverrides(
			testctx.New(),
			config.Build{
				BuildDetails: config.BuildDetails{
					Ldflags: []string{"original"},
				},
				BuildDetailsOverrides: []config.BuildDetailsOverride{
					{
						Goos:   "linux",
						Goarch: "mips",
						BuildDetails: config.BuildDetails{
							Ldflags: []string{"overridden"},
						},
					},
				},
			}, mustParse(t, "linux_mips_hardfloat"),
		)
		require.NoError(t, err)
		require.Equal(t, config.BuildDetails{
			Ldflags: []string{"overridden"},
			Env:     []string{},
		}, dets)
	})

	t.Run("with goriscv64", func(t *testing.T) {
		dets, err := withOverrides(
			testctx.New(),
			config.Build{
				BuildDetails: config.BuildDetails{
					Ldflags: []string{"original"},
				},
				BuildDetailsOverrides: []config.BuildDetailsOverride{
					{
						Goos:      "linux",
						Goarch:    "riscv64",
						Goriscv64: "rva22u64",
						BuildDetails: config.BuildDetails{
							Ldflags: []string{"overridden"},
						},
					},
				},
			}, mustParse(t, "linux_riscv64_rva22u64"),
		)
		require.NoError(t, err)
		require.Equal(t, config.BuildDetails{
			Ldflags: []string{"overridden"},
			Env:     []string{},
		}, dets)
	})

	t.Run("with goriscv64 unspecified", func(t *testing.T) {
		dets, err := withOverrides(
			testctx.New(),
			config.Build{
				BuildDetails: config.BuildDetails{
					Ldflags: []string{"original"},
				},
				BuildDetailsOverrides: []config.BuildDetailsOverride{
					{
						Goos:      "linux",
						Goarch:    "riscv64",
						Goriscv64: "rva22u64",
						BuildDetails: config.BuildDetails{
							Ldflags: []string{"overridden"},
						},
					},
				},
			}, mustParse(t, "linux_riscv64_rva22u64"),
		)
		require.NoError(t, err)
		require.Equal(t, config.BuildDetails{
			Ldflags: []string{"overridden"},
			Env:     []string{},
		}, dets)
	})

	t.Run("with go386", func(t *testing.T) {
		dets, err := withOverrides(
			testctx.New(),
			config.Build{
				BuildDetails: config.BuildDetails{
					Ldflags: []string{"original"},
				},
				BuildDetailsOverrides: []config.BuildDetailsOverride{
					{
						Goos:   "linux",
						Goarch: "386",
						Go386:  "sse2",
						BuildDetails: config.BuildDetails{
							Ldflags: []string{"overridden"},
						},
					},
				},
			}, mustParse(t, "linux_386_sse2"),
		)
		require.NoError(t, err)
		require.Equal(t, config.BuildDetails{
			Ldflags: []string{"overridden"},
			Env:     []string{},
		}, dets)
	})

	t.Run("with go386 unspecified", func(t *testing.T) {
		dets, err := withOverrides(
			testctx.New(),
			config.Build{
				BuildDetails: config.BuildDetails{
					Ldflags: []string{"original"},
				},
				BuildDetailsOverrides: []config.BuildDetailsOverride{
					{
						Goos:   "linux",
						Goarch: "386",
						Go386:  "sse2",
						BuildDetails: config.BuildDetails{
							Ldflags: []string{"overridden"},
						},
					},
				},
			}, mustParse(t, "linux_386_sse2"),
		)
		require.NoError(t, err)
		require.Equal(t, config.BuildDetails{
			Ldflags: []string{"overridden"},
			Env:     []string{},
		}, dets)
	})
}

func TestWarnIfTargetsAndOtherOptionsTogether(t *testing.T) {
	nonEmpty := []string{"foo", "bar"}
	for name, fn := range map[string]func(*config.Build){
		"goos":    func(b *config.Build) { b.Goos = nonEmpty },
		"goamd64": func(b *config.Build) { b.Goamd64 = nonEmpty },
		"goarch":  func(b *config.Build) { b.Goarch = nonEmpty },
		"goarm":   func(b *config.Build) { b.Goarm = nonEmpty },
		"gomips":  func(b *config.Build) { b.Gomips = nonEmpty },
		"ignores": func(b *config.Build) { b.Ignore = []config.IgnoredBuild{{Goos: "linux"}} },
		"multiple": func(b *config.Build) {
			b.Goos = nonEmpty
			b.Goarch = nonEmpty
			b.Goamd64 = nonEmpty
			b.Go386 = nonEmpty
			b.Goarm = nonEmpty
			b.Goarm64 = nonEmpty
			b.Gomips = nonEmpty
			b.Goppc64 = nonEmpty
			b.Goriscv64 = nonEmpty
			b.Ignore = []config.IgnoredBuild{{Goos: "linux"}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			b := config.Build{
				Targets: nonEmpty,
			}
			fn(&b)
			require.True(t, warnIfTargetsAndOtherOptionTogether(b))
		})
	}
}

func TestInvalidGoBinaryTpl(t *testing.T) {
	folder := testlib.Mktmp(t)
	require.NoError(t, os.Mkdir(filepath.Join(folder, ".go"), 0o755))
	writeGoodMain(t, folder)
	ctx := testctx.NewWithCfg(config.Project{
		Builds: []config.Build{
			{
				Targets: []string{runtimeTarget},
				Tool:    "{{.Foo}}",
				Command: "build",
			},
		},
	})
	build := ctx.Config.Builds[0]
	testlib.RequireTemplateError(t, Default.Build(ctx, build, api.Options{
		Target: mustParse(t, runtimeTarget),
		Name:   build.Binary,
		Path:   filepath.Join("dist", runtimeTarget, build.Binary),
		Ext:    "",
	}))
}

func TestBuildOutput(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		require.Empty(t, buildOutput([]byte{}))
	})
	t.Run("downloading only", func(t *testing.T) {
		require.Empty(t, buildOutput([]byte(`
go: downloading github.com/atotto/clipboard v0.1.4
go: downloading github.com/caarlos0/duration v0.0.0-20240108180406-5d492514f3c7
		`)))
	})
	t.Run("mixed", func(t *testing.T) {
		require.NotEmpty(t, buildOutput([]byte(`
go: downloading github.com/atotto/clipboard v0.1.4
go: downloading github.com/caarlos0/duration v0.0.0-20240108180406-5d492514f3c7
something something
		`)))
	})
}

func TestArtifactType(t *testing.T) {
	t.Run("c-archive", func(t *testing.T) {
		require.Equal(t, artifact.CArchive, artifactType(Target{}, "c-archive"))
	})
	t.Run("c-shared", func(t *testing.T) {
		require.Equal(t, artifact.CShared, artifactType(Target{Target: "linux_arm64"}, "c-shared"))
	})
	t.Run("c-shared/wasm", func(t *testing.T) {
		require.Equal(t, artifact.Binary, artifactType(Target{Target: "wasm"}, "c-shared"))
	})
	t.Run("binary", func(t *testing.T) {
		require.Equal(t, artifact.Binary, artifactType(Target{}, ""))
	})
}

func TestGetHeaderArtifactForLibrary(t *testing.T) {
	t.Run("no .h", func(t *testing.T) {
		require.Nil(t, getHeaderArtifactForLibrary(config.Build{}, api.Options{
			Path:   "foo.so",
			Target: Target{},
		}))
	})
	t.Run(".h", func(t *testing.T) {
		tmp := t.TempDir()
		so := filepath.Join(tmp, "foo.so")
		h := filepath.Join(tmp, "foo.h")
		require.NoError(t, os.WriteFile(so, nil, 0o644))
		require.NoError(t, os.WriteFile(h, nil, 0o644))
		a := getHeaderArtifactForLibrary(config.Build{}, api.Options{
			Path:   so,
			Ext:    ".so",
			Target: Target{},
		})
		require.NotNil(t, a)
		require.Equal(t, h, a.Path)
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

func writeTest(t *testing.T, folder string) {
	t.Helper()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "main_test.go"),
		[]byte("package main\nimport\"testing\"\nfunc TestFoo(t *testing.T) {t.Log(\"OK\")}"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "go.mod"),
		[]byte("module foo\n"),
		0o666,
	))
}

func mustParse(tb testing.TB, target string) Target {
	tb.Helper()
	got, err := Default.Parse(target)
	require.NoError(tb, err)
	return got.(Target)
}
