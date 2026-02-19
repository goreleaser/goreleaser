//go:build integration

package golang

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	api "github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestIntegrationBuild(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

func TestIntegrationBuildInvalidEnv(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

func TestIntegrationBuildCodeInSubdir(t *testing.T) {
	folder := testlib.Mktmp(t)
	subdir := filepath.Join(folder, "bar")
	err := os.Mkdir(subdir, 0o755)
	require.NoError(t, err)
	writeGoodMain(t, subdir)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

func TestIntegrationBuildWithDotGoDir(t *testing.T) {
	folder := testlib.Mktmp(t)
	require.NoError(t, os.Mkdir(filepath.Join(folder, ".go"), 0o755))
	writeGoodMain(t, folder)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

func TestIntegrationBuildFailed(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

func TestIntegrationRunInvalidAsmflags(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

func TestIntegrationRunInvalidGcflags(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

func TestIntegrationRunInvalidLdflags(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

func TestIntegrationRunInvalidFlags(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

func TestIntegrationRunPipeWithoutMainFunc(t *testing.T) {
	newCtx := func(t *testing.T) *context.Context {
		t.Helper()
		folder := testlib.Mktmp(t)
		writeMainWithoutMainFunc(t, folder)
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

func TestIntegrationBuildTests(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeTest(t, folder)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

func TestIntegrationRunPipeWithProxiedRepo(t *testing.T) {
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

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

func TestIntegrationRunPipeWithMainFuncNotInMainGoFile(t *testing.T) {
	folder := testlib.Mktmp(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "foo.go"),
		[]byte("package main\nfunc main() {println(0)}"),
		0o644,
	))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

func TestIntegrationBuildModTimestamp(t *testing.T) {
	modTime := time.Now().AddDate(-1, 0, 0).Round(time.Second).UTC()

	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)

	ctx := testctx.WrapWithCfg(t.Context(),
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
		testctx.WithVersion("5.6.7"))

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

func TestIntegrationInvalidGoBinaryTpl(t *testing.T) {
	folder := testlib.Mktmp(t)
	require.NoError(t, os.Mkdir(filepath.Join(folder, ".go"), 0o755))
	writeGoodMain(t, folder)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
