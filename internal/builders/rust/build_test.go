package rust

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
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	api "github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestAllowConcurrentBuilds(t *testing.T) {
	require.False(t, Default.AllowConcurrentBuilds())
}

func TestDependencies(t *testing.T) {
	require.NotEmpty(t, Default.Dependencies())
}

func TestWithDefaults(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		build, err := Default.WithDefaults(config.Build{})
		require.NoError(t, err)
		require.Equal(t, config.Build{
			Tool:    "cargo",
			Command: "zigbuild",
			Dir:     ".",
			Targets: defaultTargets(),
			BuildDetails: config.BuildDetails{
				Flags: []string{"--release"},
			},
		}, build)
	})

	t.Run("invalid target", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Targets: []string{"a-b"},
		})
		require.Error(t, err)
	})

	t.Run("invalid config option", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Main: "something",
		})
		require.Error(t, err)
	})
}

func TestBuild(t *testing.T) {
	testlib.CheckPath(t, "cargo")
	testlib.CheckPath(t, "cargo-zigbuild")
	folder := testlib.Mktmp(t)
	_, err := exec.Command("cargo", "init", "--bin", "--name=proj").CombinedOutput()
	require.NoError(t, err)

	modTime := time.Now().AddDate(-1, 0, 0).Round(time.Second).UTC()
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        "dist",
		ProjectName: "proj",
		Builds: []config.Build{
			{
				ID:           "default",
				Dir:          ".",
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				BuildDetails: config.BuildDetails{
					Flags: []string{"--release"},
				},
			},
		},
	})
	build, err := Default.WithDefaults(ctx.Config.Builds[0])
	require.NoError(t, err)
	require.NoError(t, Default.Prepare(ctx, build))

	target := runtimeTarget()
	if target == "" {
		t.Skip("runtime not supported")
	}

	options := api.Options{
		Name: "proj" + maybeExe(target),
		Path: filepath.Join("dist", "proj-"+target, "proj") + maybeExe(target),
		Ext:  maybeExe(target),
	}
	options.Target, err = Default.Parse(target)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(options.Path), 0o755)) // this happens on internal/pipe/build/ when in prod

	require.NoError(t, Default.Build(ctx, build, options))

	list := ctx.Artifacts
	require.NoError(t, list.Visit(func(a *artifact.Artifact) error {
		s, err := filepath.Rel(folder, a.Path)
		if err == nil {
			a.Path = s
		}
		return nil
	}))

	bins := list.List()
	require.Len(t, bins, 1)

	bin := bins[0]
	require.Equal(t, artifact.Artifact{
		Name:   "proj" + maybeExe(target),
		Path:   filepath.ToSlash(options.Path),
		Goos:   runtime.GOOS,
		Goarch: runtime.GOARCH,
		Target: target,
		Type:   artifact.Binary,
		Extra: artifact.Extras{
			artifact.ExtraBinary:  "proj",
			artifact.ExtraBuilder: "rust",
			artifact.ExtraExt:     maybeExe(target),
			artifact.ExtraID:      "default",
		},
	}, *bin)

	require.FileExists(t, bin.Path)
	fi, err := os.Stat(bin.Path)
	require.NoError(t, err)
	require.True(t, modTime.Equal(fi.ModTime()))
}

func TestParse(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		_, err := Default.Parse("a-b")
		require.Error(t, err)
	})

	t.Run("triplet", func(t *testing.T) {
		target, err := Default.Parse("aarch64-apple-darwin")
		require.NoError(t, err)
		require.Equal(t, Target{
			Target: "aarch64-apple-darwin",
			Os:     "darwin",
			Arch:   "arm64",
			Vendor: "apple",
		}, target)
	})

	t.Run("quadruplet", func(t *testing.T) {
		target, err := Default.Parse("aarch64-pc-windows-gnullvm")
		require.NoError(t, err)
		require.Equal(t, Target{
			Target:      "aarch64-pc-windows-gnullvm",
			Os:          "windows",
			Arch:        "arm64",
			Vendor:      "pc",
			Environment: "gnullvm",
		}, target)
	})
}

func TestIsSettingPackage(t *testing.T) {
	for name, tt := range map[string]struct {
		flags  []string
		expect bool
	}{
		"not set":   {[]string{"--release", "--something-else", "--in-the-p=middle", "--something"}, false},
		"-p":        {[]string{"--release", "-p=foo", "--something"}, true},
		"--package": {[]string{"--release", "--package=foo", "--something"}, true},
	} {
		t.Run(name, func(t *testing.T) {
			got := isSettingPackage(tt.flags)
			require.Equal(t, tt.expect, got)
		})
	}
}

func runtimeTarget() string {
	targets := map[string]string{
		"windows-arm64": "aarch64-pc-windows-msvc",
		"linux-amd64":   "x86_64-unknown-linux-gnu",
		"linux-arm64":   "aarch64-unknown-linux-gnu",
		"darwin-amd64":  "x86_64-apple-darwin",
		"darwin-arm64":  "aarch64-apple-darwin",
	}
	return targets[runtime.GOOS+"-"+runtime.GOARCH]
}

func maybeExe(s string) string {
	if strings.Contains(s, "windows") {
		return ".exe"
	}
	return ""
}
