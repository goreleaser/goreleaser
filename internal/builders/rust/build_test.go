package rust

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func TestCustomGlibc(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Targets: []string{"aarch64-unknown-linux-gnu.2.17"},
		})
		require.NoError(t, err)
	})
	t.Run("valid-gnueabihf", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Targets: []string{"armv7-unknown-linux-gnueabihf.2.17"},
		})
		require.NoError(t, err)
	})
	t.Run("invalid", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Targets: []string{"aarch64-unknown-linux-musl.2.17"},
		})
		require.ErrorContains(t, err, "invalid target")
	})
	t.Run("invalid-gnullvm", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Targets: []string{"aarch64-pc-windows-gnullvm.2.17"},
		})
		require.ErrorContains(t, err, "invalid target")
	})
}

func TestBuildWorkspaceErrorShowsAllMembers(t *testing.T) {
	dir := testlib.Mktmp(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(`
[workspace]
members = ["crate-a", "crate-b", "crate-c"]
`), 0o644))

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist: "dist",
		Builds: []config.Build{{
			Dir:          ".",
			BuildDetails: config.BuildDetails{Flags: []string{"--release"}},
		}},
	})

	target, err := Default.Parse("aarch64-unknown-linux-gnu")
	require.NoError(t, err)

	err = Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Name:   "proj",
		Path:   "dist/proj",
		Target: target,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "crate-a")
	require.Contains(t, err.Error(), "crate-b")
	require.Contains(t, err.Error(), "crate-c")
}

func TestBuild(t *testing.T) {
	testlib.CheckPath(t, "cargo")
	testlib.CheckPath(t, "cargo-zigbuild")

	folder := testlib.Mktmp(t)
	_, err := exec.CommandContext(t.Context(), "cargo", "init", "--bin", "--name=proj").CombinedOutput()
	require.NoError(t, err)

	f, err := os.OpenFile("Cargo.toml", os.O_APPEND|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	_, err = f.WriteString("\n[profile.release]\nopt-level = 0\n")
	require.NoError(t, f.Close())
	require.NoError(t, err)

	target := "aarch64-unknown-linux-gnu.2.17"
	modTime := time.Now().AddDate(-1, 0, 0).Round(time.Second).UTC()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        "dist",
		ProjectName: "proj",
		Builds: []config.Build{
			{
				ID:           "default",
				Dir:          ".",
				Targets:      []string{target},
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

	options := api.Options{
		Name: "proj",
		Path: filepath.Join("dist", "proj-"+target, "proj"),
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
		Name:   "proj",
		Path:   filepath.ToSlash(options.Path),
		Goos:   "linux",
		Goarch: "arm64",
		Target: target,
		Type:   artifact.Binary,
		Extra: artifact.Extras{
			artifact.ExtraBinary:   "proj",
			artifact.ExtraBuilder:  "rust",
			artifact.ExtraExt:      "",
			artifact.ExtraID:       "default",
			artifact.ExtranDynLink: true,
			keyAbi:                 "gnu",
			keyLibc:                "2.17",
		},
	}, *bin)

	require.FileExists(t, bin.Path)
	fi, err := os.Stat(bin.Path)
	require.NoError(t, err)
	require.True(t, modTime.Equal(fi.ModTime()))
}

func TestBuildArm(t *testing.T) {
	testlib.CheckPath(t, "cargo")
	testlib.CheckPath(t, "cargo-zigbuild")

	folder := testlib.Mktmp(t)
	_, err := exec.CommandContext(t.Context(), "cargo", "init", "--bin", "--name=proj").CombinedOutput()
	require.NoError(t, err)

	f, err := os.OpenFile("Cargo.toml", os.O_APPEND|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	_, err = f.WriteString("\n[profile.release]\nopt-level = 0\n")
	require.NoError(t, f.Close())
	require.NoError(t, err)

	target := "armv7-unknown-linux-gnueabihf.2.17"
	modTime := time.Now().AddDate(-1, 0, 0).Round(time.Second).UTC()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        "dist",
		ProjectName: "proj",
		Builds: []config.Build{
			{
				ID:           "default",
				Dir:          ".",
				Targets:      []string{target},
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

	options := api.Options{
		Name: "proj",
		Path: filepath.Join("dist", "proj-"+target, "proj"),
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
		Name:   "proj",
		Path:   filepath.ToSlash(options.Path),
		Goos:   "linux",
		Goarch: "arm",
		Goarm:  "7",
		Target: target,
		Type:   artifact.Binary,
		Extra: artifact.Extras{
			artifact.ExtraBinary:   "proj",
			artifact.ExtraBuilder:  "rust",
			artifact.ExtraExt:      "",
			artifact.ExtraID:       "default",
			artifact.ExtranDynLink: true,
			keyAbi:                 "gnueabihf",
			keyLibc:                "2.17",
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
			Target: "aarch64-pc-windows-gnullvm",
			Os:     "windows",
			Arch:   "arm64",
			Vendor: "pc",
			Abi:    "gnullvm",
		}, target)
	})
	t.Run("glibc-version", func(t *testing.T) {
		target, err := Default.Parse("aarch64-unknown-linux-gnu.2.17")
		require.NoError(t, err)
		require.Equal(t, Target{
			Target: "aarch64-unknown-linux-gnu.2.17",
			Os:     "linux",
			Arch:   "arm64",
			Vendor: "unknown",
			Abi:    "gnu",
			Libc:   "2.17",
		}, target)
	})
	t.Run("glibc-version-gnueabihf", func(t *testing.T) {
		target, err := Default.Parse("armv7-unknown-linux-gnueabihf.2.17")
		require.NoError(t, err)
		require.Equal(t, Target{
			Target: "armv7-unknown-linux-gnueabihf.2.17",
			Os:     "linux",
			Arch:   "arm",
			Arm:    "7",
			Vendor: "unknown",
			Abi:    "gnueabihf",
			Libc:   "2.17",
		}, target)
	})
}

func TestStripGlibcVersion(t *testing.T) {
	for name, tt := range map[string]struct {
		input string
		want  string
		ok    bool
	}{
		"gnu":       {"aarch64-unknown-linux-gnu.2.17", "aarch64-unknown-linux-gnu", true},
		"gnueabihf": {"armv7-unknown-linux-gnueabihf.2.17", "armv7-unknown-linux-gnueabihf", true},
		"gnueabi":   {"arm-unknown-linux-gnueabi.2.31", "arm-unknown-linux-gnueabi", true},
		"no-suffix": {"aarch64-unknown-linux-gnu", "aarch64-unknown-linux-gnu", false},
		"musl":      {"aarch64-unknown-linux-musl.2.17", "aarch64-unknown-linux-musl.2.17", false},
		"gnullvm":   {"aarch64-pc-windows-gnullvm.2.17", "aarch64-pc-windows-gnullvm.2.17", false},
		"no-dashes": {"nodashes.2.17", "nodashes.2.17", false},
	} {
		t.Run(name, func(t *testing.T) {
			got, ok := stripGlibcVersion(tt.input)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.want, got)
		})
	}
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
