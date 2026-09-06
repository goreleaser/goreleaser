package buildtarget_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/builders/buildtarget"
	"github.com/goreleaser/goreleaser/v2/internal/builders/rust"
	"github.com/goreleaser/goreleaser/v2/internal/builders/zig"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	api "github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestCanonicalArchitectures(t *testing.T) {
	tool := buildFakeTool(t)
	for name, tt := range map[string]struct {
		builder    api.Builder
		builderID  string
		target     string
		sourceArch string
		wantGoos   string
		wantGoarch string
		command    string
		flags      []string
		prepare    func(*testing.T, string)
	}{
		"rust riscv64gc": {
			builder:    rust.Default,
			builderID:  "rust",
			target:     "riscv64gc-unknown-linux-gnu",
			sourceArch: "riscv64gc",
			wantGoos:   "linux",
			wantGoarch: "riscv64",
			command:    "zigbuild",
			flags:      []string{"--release"},
			prepare:    writeCargoProject,
		},
		"rust loongarch64": {
			builder:    rust.Default,
			builderID:  "rust",
			target:     "loongarch64-unknown-linux-gnu",
			sourceArch: "loongarch64",
			wantGoos:   "linux",
			wantGoarch: "loong64",
			command:    "zigbuild",
			flags:      []string{"--release"},
			prepare:    writeCargoProject,
		},
		"zig x86": {
			builder:    zig.Default,
			builderID:  "zig",
			target:     "x86-linux-gnu",
			sourceArch: "x86",
			wantGoos:   "linux",
			wantGoarch: "386",
			command:    "build",
			flags:      []string{"-Doptimize=ReleaseSafe"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tt.wantGoarch, buildtarget.Goarch(tt.sourceArch))

			target, err := tt.builder.Parse(tt.target)
			require.NoError(t, err)
			require.Equal(t, tt.target, target.String())
			require.Equal(t, tt.wantGoarch, target.Fields()[tmpl.KeyArch])

			rendered, err := tmpl.New(testctx.Wrap(t.Context())).
				WithBuildOptions(api.Options{Target: target}).
				Apply("{{ .Target }} {{ .Arch }}")
			require.NoError(t, err)
			require.Equal(t, tt.target+" "+tt.wantGoarch, rendered)

			dir := t.TempDir()
			if tt.prepare != nil {
				tt.prepare(t, dir)
			}
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Dist: filepath.Join(dir, "dist"),
				Env: []string{
					"GORELEASER_FAKE_BINARY=proj",
				},
				Builds: []config.Build{{
					ID:      "default",
					Builder: tt.builderID,
					Dir:     dir,
					Targets: []string{tt.target},
					Tool:    tool,
					Command: tt.command,
					Flags:   tt.flags,
				}},
			})

			build, err := tt.builder.WithDefaults(ctx.Config.Builds[0])
			require.NoError(t, err)

			options := api.Options{
				Name:   "proj",
				Path:   filepath.Join(ctx.Config.Dist, "proj"),
				Target: target,
			}
			require.NoError(t, os.MkdirAll(filepath.Dir(options.Path), 0o755))

			require.NoError(t, tt.builder.Build(ctx, build, options))

			binaries := ctx.Artifacts.List()
			require.Len(t, binaries, 1)
			require.Equal(t, tt.wantGoos, binaries[0].Goos)
			require.Equal(t, tt.wantGoarch, binaries[0].Goarch)
			require.Equal(t, tt.target, binaries[0].Target)
			require.Len(t, ctx.Artifacts.Filter(artifact.ByGoarch(tt.wantGoarch)).List(), 1)
			require.Empty(t, ctx.Artifacts.Filter(artifact.ByGoarch(tt.sourceArch)).List())
		})
	}
}

func writeCargoProject(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(`
[package]
name = "proj"
version = "0.1.0"
edition = "2021"
`), 0o644))
}

func buildFakeTool(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(source, []byte(`package main

import (
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var target, prefix string
	args := os.Args[1:]
	for i, arg := range args {
		if strings.HasPrefix(arg, "--target=") {
			target = strings.TrimPrefix(arg, "--target=")
		}
		if strings.HasPrefix(arg, "-Dtarget=") {
			target = strings.TrimPrefix(arg, "-Dtarget=")
		}
		if arg == "-p" && i+1 < len(args) {
			prefix = args[i+1]
		}
	}

	name := os.Getenv("GORELEASER_FAKE_BINARY")
	if name == "" {
		panic("GORELEASER_FAKE_BINARY is required")
	}

	var path string
	if prefix == "" {
		if target == "" {
			panic("target is required")
		}
		clean, _, _ := strings.Cut(target, ".")
		path = filepath.Join("target", clean, "release", name)
	} else {
		path = filepath.Join(prefix, "bin", name)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, []byte("fake"), 0o755); err != nil {
		panic(err)
	}
}
`), 0o644))

	tool := filepath.Join(dir, "fake-builder")
	if runtime.GOOS == "windows" {
		tool += ".exe"
	}
	cmd := exec.CommandContext(t.Context(), "go", "build", "-o", tool, source)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return tool
}
