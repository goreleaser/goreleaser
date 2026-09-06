package flatpak

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestContinueOnError(t *testing.T) {
	require.True(t, Pipe{}.ContinueOnError())
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
	})
	t.Run("skip flag", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Flatpaks: []config.Flatpak{{}},
		}, testctx.Skip(skips.Flatpak))
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Flatpaks: []config.Flatpak{{}},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Flatpaks: []config.Flatpak{validFlatpak()},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, defaultNameTemplate, ctx.Config.Flatpaks[0].NameTemplate)
}

func TestDefaultMissingFields(t *testing.T) {
	for name, mod := range map[string]func(*config.Flatpak){
		"no app_id":          func(fp *config.Flatpak) { fp.AppID = "" },
		"no runtime":         func(fp *config.Flatpak) { fp.Runtime = "" },
		"no runtime_version": func(fp *config.Flatpak) { fp.RuntimeVersion = "" },
		"no sdk":             func(fp *config.Flatpak) { fp.SDK = "" },
	} {
		t.Run(name, func(t *testing.T) {
			fp := validFlatpak()
			mod(&fp)
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Flatpaks: []config.Flatpak{fp},
			})
			require.Error(t, Pipe{}.Default(ctx))
		})
	}
}

func TestSeveralFlatpaksWithTheSameID(t *testing.T) {
	fp1 := validFlatpak()
	fp1.ID = "a"
	fp2 := validFlatpak()
	fp2.ID = "a"
	fp2.AppID = "org.example.App2"
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Flatpaks: []config.Flatpak{fp1, fp2},
	})
	require.EqualError(t, Pipe{}.Default(ctx), "found 2 flatpaks with the ID 'a', please fix your config")
}

func TestNoFlatpakBuilderInPath(t *testing.T) {
	t.Setenv("PATH", "")
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Flatpaks: []config.Flatpak{validFlatpak()},
	})
	require.ErrorIs(t, Pipe{}.Run(ctx), ErrNoFlatpakBuilder)
}

func TestRunPipeDisabled(t *testing.T) {
	fp := validFlatpak()
	fp.Disable = "true"
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Flatpaks: []config.Flatpak{fp},
	})
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestRunPipeDisabledTemplate(t *testing.T) {
	fp := validFlatpak()
	fp.Disable = "{{.Env.SKIP}}"
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Flatpaks: []config.Flatpak{fp},
	}, testctx.WithEnv(map[string]string{"SKIP": "true"}))
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestRunPipeSomeDisabled(t *testing.T) {
	testlib.OnlyOnLinux(t, "flatpak only works on linux")
	testlib.CheckPath(t, "flatpak-builder")
	testlib.CheckPath(t, "flatpak")
	dist := filepath.Join(t.TempDir(), "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))

	disabled := validFlatpak()
	disabled.ID = "disabled"
	disabled.Disable = "true"

	enabled := validFlatpak()
	enabled.ID = "enabled"
	enabled.NameTemplate = "foo_{{.Arch}}"
	enabled.AppID = "org.example.MyBin"
	enabled.IDs = []string{"foo"}

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Flatpaks:    []config.Flatpak{disabled, enabled},
	}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"))

	addBinaries(t, ctx, "foo", filepath.Join(dist, "foo"))
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))

	list := ctx.Artifacts.Filter(artifact.ByType(artifact.Flatpak)).List()
	require.NotEmpty(t, list, "the flatpak after the disabled one should have been built")
	require.Equal(t, "enabled", artifact.ExtraOr(*list[0], artifact.ExtraID, ""))
}

func TestRunPipeInvalidNameTemplate(t *testing.T) {
	testlib.OnlyOnLinux(t, "flatpak only works on linux")
	testlib.CheckPath(t, "flatpak-builder")
	dist := filepath.Join(t.TempDir(), "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	fp := validFlatpak()
	fp.NameTemplate = "foo_{{.Arch}"
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "foo",
		Dist:        dist,
		Flatpaks:    []config.Flatpak{fp},
	}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"))

	addBinaries(t, ctx, "foo", dist)
	testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
}

func TestRunPipe(t *testing.T) {
	testlib.OnlyOnLinux(t, "flatpak only works on linux")
	testlib.CheckPath(t, "flatpak-builder")
	testlib.CheckPath(t, "flatpak")
	dist := filepath.Join(t.TempDir(), "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	fp := validFlatpak()
	fp.NameTemplate = "foo_{{.Arch}}"
	fp.AppID = "org.example.MyBin"
	fp.IDs = []string{"foo"}
	fp.Command = "foo"
	fp.FinishArgs = []string{"--share=network", "--socket=x11"}
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Flatpaks:    []config.Flatpak{fp},
	}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"))

	addBinaries(t, ctx, "foo", filepath.Join(dist, "foo"))
	requireNoGerror(t, Pipe{}.Run(ctx))

	list := ctx.Artifacts.Filter(artifact.ByType(artifact.Flatpak)).List()
	require.NotEmpty(t, list)

	manifestFile := filepath.Join(dist, "flatpak", "foo_amd64", "x86_64", "org.example.MyBin.json")
	manifestBytes, err := os.ReadFile(manifestFile)
	require.NoError(t, err)

	var manifest Manifest
	require.NoError(t, json.Unmarshal(manifestBytes, &manifest))
	require.Equal(t, "org.example.MyBin", manifest.ID)
	require.Equal(t, "org.freedesktop.Platform", manifest.Runtime)
	require.Equal(t, "24.08", manifest.RuntimeVersion)
	require.Equal(t, "org.freedesktop.Sdk", manifest.SDK)
	require.Equal(t, "foo", manifest.Command)
	require.Equal(t, []string{"--share=network", "--socket=x11"}, manifest.FinishArgs)
	require.Len(t, manifest.Modules, 1)
	require.Equal(t, "simple", manifest.Modules[0].BuildSystem)
}

func TestRunPipeQuotesInstallPaths(t *testing.T) {
	testlib.SkipIfWindows(t, "flatpak build commands are shell commands")

	for name, binaryName := range map[string]string{
		"ordinary": "myapp",
		"spaces":   "my app",
		"quotes":   `my "app's"`,
		"hostile":  "my 'app\" with space \\ $HOME `whoami` $(id)\nnext",
	} {
		t.Run(name, func(t *testing.T) {
			binDir := t.TempDir()
			writeFlatpakTestCommand(t, binDir, "flatpak-builder")
			writeFlatpakTestCommand(t, binDir, "flatpak")
			writeFlatpakTestCommand(t, binDir, "install")
			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("GO_WANT_FLATPAK_COMMAND_HELPER", "1")
			t.Setenv("EXPECTED_INSTALL_SOURCE", binaryName)
			t.Setenv("EXPECTED_INSTALL_DEST", "/app/bin/"+binaryName)
			record := filepath.Join(t.TempDir(), "install-record")
			t.Setenv("INSTALL_RECORD", record)

			dist := filepath.Join(t.TempDir(), "dist")
			require.NoError(t, os.Mkdir(dist, 0o755))
			fp := validFlatpak()
			fp.NameTemplate = "foo_{{.Arch}}"
			fp.AppID = "org.example.MyBin"
			fp.IDs = []string{binaryName}
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				ProjectName: "mybin",
				Dist:        dist,
				Flatpaks:    []config.Flatpak{fp},
			}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"))

			addBinaries(t, ctx, binaryName, dist)
			requireNoGerror(t, Pipe{}.Run(ctx))

			contents, err := os.ReadFile(record)
			require.NoError(t, err)
			require.Contains(t, string(contents), binaryName)
			require.Contains(t, string(contents), "/app/bin/"+binaryName)
		})
	}
}

func TestRunCmdEnvPrecedence(t *testing.T) {
	for name, tt := range map[string]struct {
		ambient string
		project string
		want    string
	}{
		"conflicting": {
			ambient: "ambient-config",
			project: "project-config",
			want:    "project-config",
		},
		"configured-only": {
			project: "project-config",
			want:    "project-config",
		},
		"inherited-only": {
			ambient: "ambient-config",
			want:    "ambient-config",
		},
	} {
		t.Run(name, func(t *testing.T) {
			key := "GORELEASER_TEST_FLATPAK_ENV_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
			unsetEnv(t, key)
			if tt.ambient != "" {
				t.Setenv(key, tt.ambient)
			}
			t.Setenv("GO_WANT_FLATPAK_COMMAND_HELPER", "1")
			t.Setenv("FLATPAK_ENV_HELPER_KEY", key)
			record := filepath.Join(t.TempDir(), "record")
			t.Setenv("FLATPAK_ENV_HELPER_RECORD", record)

			cfg := config.Project{}
			if tt.project != "" {
				cfg.Env = []string{key + "=" + tt.project}
			}
			ctx := testctx.WrapWithCfg(t.Context(), cfg)
			require.NoError(t, runCmd(
				ctx,
				t.TempDir(),
				"failed to run flatpak helper",
				os.Args[0],
				"-test.run=TestFlatpakCommandHelper",
				"--",
				"env",
			))

			bts, err := os.ReadFile(record)
			require.NoError(t, err)
			require.Equal(t, tt.want, string(bts))
		})
	}
}

func TestDependencies(t *testing.T) {
	require.Equal(t, []string{"flatpak-builder", "flatpak"}, Pipe{}.Dependencies(nil))
}

func validFlatpak() config.Flatpak {
	return config.Flatpak{
		AppID:          "org.example.App",
		Runtime:        "org.freedesktop.Platform",
		RuntimeVersion: "24.08",
		SDK:            "org.freedesktop.Sdk",
	}
}

func requireNoGerror(tb testing.TB, err error) {
	tb.Helper()
	if err == nil {
		return
	}
	gerr, ok := errors.AsType[gerrors.ErrDetailed](err)
	require.True(tb, ok)
	require.NoError(tb, err, "messages: %v, details: %v, output: %s", gerr.Messages(), maps.Collect(gerr.Details()), gerr.Output())
}

func addBinaries(t *testing.T, ctx *context.Context, name, dist string) {
	t.Helper()
	binPath := filepath.Join(dist, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(binPath), 0o755))
	f, err := os.Create(binPath)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "arm64"} {
			a := &artifact.Artifact{
				Name:   name,
				Path:   binPath,
				Goarch: goarch,
				Goos:   goos,
				Type:   artifact.Binary,
				Extra: map[string]any{
					artifact.ExtraID: name,
				},
			}
			if goarch == "amd64" {
				a.Goamd64 = "v1"
			}
			ctx.Artifacts.Add(a)
		}
	}
}

func writeFlatpakTestCommand(t *testing.T, dir, name string) {
	t.Helper()
	script := fmt.Sprintf("#!/bin/sh\nexec %q -test.run=TestFlatpakCommandHelper -- %q \"$@\"\n", os.Args[0], name)
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755))
	if testlib.IsWindows() {
		script := fmt.Sprintf("@echo off\r\n%q -test.run=TestFlatpakCommandHelper -- %q %%*\r\n", os.Args[0], name)
		require.NoError(t, os.WriteFile(filepath.Join(dir, name+".bat"), []byte(script), 0o755))
	}
}

func TestFlatpakCommandHelper(_ *testing.T) {
	if os.Getenv("GO_WANT_FLATPAK_COMMAND_HELPER") != "1" {
		return
	}

	args := flatpakCommandHelperArgs()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "missing helper command")
		os.Exit(2)
	}

	switch args[0] {
	case "flatpak-builder":
		runFlatpakBuilderHelper(args[1:])
	case "flatpak":
		os.Exit(0)
	case "install":
		runInstallHelper(args[1:])
	case "env":
		runFlatpakEnvHelper()
	default:
		fmt.Fprintf(os.Stderr, "unknown helper command: %s\n", args[0])
		os.Exit(2)
	}
}

func flatpakCommandHelperArgs() []string {
	for i, arg := range os.Args {
		if arg == "--" {
			return os.Args[i+1:]
		}
	}
	return nil
}

func runFlatpakBuilderHelper(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "missing manifest path")
		os.Exit(2)
	}

	manifestBytes, err := os.ReadFile(args[len(args)-1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "read manifest: %v\n", err)
		os.Exit(2)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		fmt.Fprintf(os.Stderr, "unmarshal manifest: %v\n", err)
		os.Exit(2)
	}

	for _, module := range manifest.Modules {
		for _, buildCommand := range module.BuildCommands {
			cmd := exec.Command("sh", "-c", buildCommand)
			cmd.Env = os.Environ()
			out, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Fprintf(os.Stderr, "run build command %q: %v\n%s", buildCommand, err, out)
				os.Exit(1)
			}
		}
	}
	os.Exit(0)
}

func runFlatpakEnvHelper() {
	value := os.Getenv(os.Getenv("FLATPAK_ENV_HELPER_KEY"))
	if err := os.WriteFile(os.Getenv("FLATPAK_ENV_HELPER_RECORD"), []byte(value), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write record: %v\n", err)
		os.Exit(2)
	}
	os.Exit(0)
}

func runInstallHelper(args []string) {
	expected := []string{
		"-Dm755",
		os.Getenv("EXPECTED_INSTALL_SOURCE"),
		os.Getenv("EXPECTED_INSTALL_DEST"),
	}
	if !slices.Equal(args, expected) {
		fmt.Fprintf(os.Stderr, "install args: got %#v, want %#v\n", args, expected)
		os.Exit(64)
	}
	f, err := os.OpenFile(os.Getenv("INSTALL_RECORD"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open install record: %v\n", err)
		os.Exit(2)
	}
	if _, err := fmt.Fprintf(f, "%s\n%s\n", args[1], args[2]); err != nil {
		fmt.Fprintf(os.Stderr, "write install record: %v\n", err)
		os.Exit(2)
	}
	if err := f.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "close install record: %v\n", err)
		os.Exit(2)
	}
	os.Exit(0)
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "")
	require.NoError(t, os.Unsetenv(key))
}
