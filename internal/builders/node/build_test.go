package node

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDependencies(t *testing.T) {
	require.Equal(t, []string{"node"}, Default.Dependencies())
}

func TestAllowConcurrentBuilds(t *testing.T) {
	require.True(t, Default.AllowConcurrentBuilds())
}

func TestParse(t *testing.T) {
	for target, dst := range map[string]Target{
		"linux-x64":    {Target: "linux-x64", Os: "linux", Arch: "x64"},
		"darwin-arm64": {Target: "darwin-arm64", Os: "darwin", Arch: "arm64"},
		"win-arm64":    {Target: "win-arm64", Os: "win", Arch: "arm64"},
	} {
		t.Run(target, func(t *testing.T) {
			got, err := Default.Parse(target)
			require.NoError(t, err)
			require.IsType(t, Target{}, got)
			require.Equal(t, dst, got.(Target))
		})
	}

	t.Run("invalid", func(t *testing.T) {
		_, err := Default.Parse("linux")
		require.Error(t, err)
	})
	t.Run("unknown target", func(t *testing.T) {
		_, err := Default.Parse("plan9-amd64")
		require.Error(t, err)
	})
}

func TestIsValid(t *testing.T) {
	for _, target := range []string{
		"darwin-arm64",
		"darwin-x64",
		"linux-arm64",
		"linux-x64",
		"win-arm64",
		"win-x64",
	} {
		t.Run(target, func(t *testing.T) {
			require.True(t, isValid(target))
		})
	}
	t.Run("invalid", func(t *testing.T) {
		require.False(t, isValid("plan9-amd64"))
	})
}

func TestWithDefaults(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		build, err := Default.WithDefaults(config.Build{})
		require.NoError(t, err)
		require.Equal(t, ".", build.Dir)
		require.Equal(t, "index.js", build.Main)
		require.ElementsMatch(t, defaultTargets(), build.Targets)
	})

	t.Run("respects main override", func(t *testing.T) {
		build, err := Default.WithDefaults(config.Build{Main: "src/cli.mjs"})
		require.NoError(t, err)
		require.Equal(t, "src/cli.mjs", build.Main)
	})

	t.Run("invalid target", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Targets: []string{"plan9-amd64"},
		})
		require.Error(t, err)
	})

	t.Run("rejects tool", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{Tool: "node"})
		require.ErrorContains(t, err, "tool is not supported")
	})

	t.Run("rejects command", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{Command: "build"})
		require.ErrorContains(t, err, "command is not supported")
	})

	t.Run("rejects flags", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			BuildDetails: config.BuildDetails{Flags: []string{"--x"}},
		})
		require.ErrorContains(t, err, "flags is not supported")
	})
}

func TestCurrentTarget(t *testing.T) {
	got := currentTarget()
	require.NotEmpty(t, got)
	// Must always parse cleanly under our own rules.
	require.True(t, isValid(got), "%s should be a valid target", got)
}

// TestRunNPMBuildScript covers the per-build npm wire-up: silent skip
// paths, error propagation, and that build.Env templating reaches the
// spawned `npm` process. The end-to-end behaviour of `npm run build`
// itself is exercised by the unit tests in `internal/nodesea`; here we
// validate the glue that turns a build config into a RunNPMBuild call.
func TestRunNPMBuildScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh fake npm")
	}

	writePackageJSON := func(t *testing.T, dir string, scripts map[string]string) {
		t.Helper()
		var sb []byte
		sb = append(sb, `{"scripts":{`...)
		first := true
		for k, v := range scripts {
			if !first {
				sb = append(sb, ',')
			}
			first = false
			sb = append(sb, '"')
			sb = append(sb, k...)
			sb = append(sb, `":"`...)
			sb = append(sb, v...)
			sb = append(sb, '"')
		}
		sb = append(sb, `}}`...)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), sb, 0o644))
	}

	// fakeNPM drops a tiny shell script at <bindir>/npm, prepends
	// bindir to PATH for the test, and writes the script body
	// supplied by the caller (typically appending args to a log file).
	fakeNPM := func(t *testing.T, body string) {
		t.Helper()
		bindir := t.TempDir()
		require.NoError(t, os.WriteFile(
			filepath.Join(bindir, "npm"),
			[]byte("#!/bin/sh\n"+body+"\n"),
			0o755,
		))
		t.Setenv("PATH", bindir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}

	t.Run("runs npm run build when scripts.build is declared", func(t *testing.T) {
		dir := t.TempDir()
		writePackageJSON(t, dir, map[string]string{"build": "esbuild ..."})
		fakeNPM(t, "echo \"$@\" >> \""+dir+"/calls.log\"\nexit 0")

		ctx := testctx.Wrap(t.Context())
		require.NoError(t, runNPMBuildScript(ctx, config.Build{Dir: dir}))

		got, err := os.ReadFile(filepath.Join(dir, "calls.log"))
		require.NoError(t, err)
		require.Equal(t, "run build\n", string(got))
	})

	t.Run("silent skip when scripts.build missing", func(t *testing.T) {
		dir := t.TempDir()
		writePackageJSON(t, dir, map[string]string{"test": "vitest"})
		ctx := testctx.Wrap(t.Context())
		require.NoError(t, runNPMBuildScript(ctx, config.Build{Dir: dir}))
	})

	t.Run("silent skip when no package.json", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		require.NoError(t, runNPMBuildScript(ctx, config.Build{Dir: t.TempDir()}))
	})

	t.Run("templates build.env and forwards to npm", func(t *testing.T) {
		dir := t.TempDir()
		writePackageJSON(t, dir, map[string]string{"build": "esbuild ..."})
		fakeNPM(t, "printf 'NODE_ENV=%s\\n' \"$NODE_ENV\" > \""+dir+"/env.log\"\nexit 0")

		ctx := testctx.Wrap(t.Context())
		ctx.Env = map[string]string{"WANTED_ENV": "production"}
		require.NoError(t, runNPMBuildScript(ctx, config.Build{
			Dir:          dir,
			BuildDetails: config.BuildDetails{Env: []string{"NODE_ENV={{ .Env.WANTED_ENV }}"}},
		}))

		got, err := os.ReadFile(filepath.Join(dir, "env.log"))
		require.NoError(t, err)
		require.Equal(t, "NODE_ENV=production\n", string(got))
	})

	t.Run("non-zero exit propagates", func(t *testing.T) {
		dir := t.TempDir()
		writePackageJSON(t, dir, map[string]string{"build": "esbuild ..."})
		fakeNPM(t, "exit 1")

		ctx := testctx.Wrap(t.Context())
		require.Error(t, runNPMBuildScript(ctx, config.Build{Dir: dir}))
	})

	t.Run("invalid template surfaces error", func(t *testing.T) {
		dir := t.TempDir()
		writePackageJSON(t, dir, map[string]string{"build": "esbuild ..."})
		ctx := testctx.Wrap(t.Context())
		err := runNPMBuildScript(ctx, config.Build{
			Dir:          dir,
			BuildDetails: config.BuildDetails{Env: []string{"X={{ .NotARealField }}"}},
		})
		require.Error(t, err)
	})
}
