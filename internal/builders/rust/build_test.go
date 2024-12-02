package rust

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
	api "github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestWithDefaults(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		build, err := Default.WithDefaults(config.Build{})
		require.NoError(t, err)
		require.Equal(t, config.Build{
			GoBinary: "cargo",
			Command:  "zigbuild",
			Dir:      ".",
			Targets:  defaultTargets(),
		}, build)
	})

	t.Run("invalid", func(t *testing.T) {
		cases := map[string]config.Build{
			"main": {
				Main: "a",
			},
			"ldflags": {
				BuildDetails: config.BuildDetails{
					Ldflags: []string{"-a"},
				},
			},
			"goos": {
				Goos: []string{"a"},
			},
			"goarch": {
				Goarch: []string{"a"},
			},
			"goamd64": {
				Goamd64: []string{"a"},
			},
			"go386": {
				Go386: []string{"a"},
			},
			"goarm": {
				Goarm: []string{"a"},
			},
			"goarm64": {
				Goarm64: []string{"a"},
			},
			"gomips": {
				Gomips: []string{"a"},
			},
			"goppc64": {
				Goppc64: []string{"a"},
			},
			"goriscv64": {
				Goriscv64: []string{"a"},
			},
			"ignore": {
				Ignore: []config.IgnoredBuild{{}},
			},
			"overrides": {
				BuildDetailsOverrides: []config.BuildDetailsOverride{{}},
			},
			"buildmode": {
				BuildDetails: config.BuildDetails{
					Buildmode: "a",
				},
			},
			"tags": {
				BuildDetails: config.BuildDetails{
					Tags: []string{"a"},
				},
			},
			"asmflags": {
				BuildDetails: config.BuildDetails{
					Asmflags: []string{"a"},
				},
			},
		}
		for k, v := range cases {
			t.Run(k, func(t *testing.T) {
				_, err := Default.WithDefaults(v)
				require.Error(t, err)
			})
		}
	})
}

func TestBuild(t *testing.T) {
	testlib.CheckPath(t, "rustup")
	testlib.CheckPath(t, "cargo")

	for _, s := range []string{
		"rustup default stable",
		"cargo install --locked cargo-zigbuild",
	} {
		args := strings.Fields(s)
		_, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		require.NoError(t, err)
	}

	modTime := time.Now().AddDate(-1, 0, 0).Round(1 * time.Second).UTC()
	dist := t.TempDir()
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        dist,
		ProjectName: "proj",
		Env: []string{
			`TEST_E=1`,
		},
		Builds: []config.Build{
			{
				ID:           "default",
				Dir:          "./testdata/proj/",
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				BuildDetails: config.BuildDetails{
					Env: []string{
						`TEST_T={{- if eq .Os "windows" -}}
							w
						{{- else if eq .Os "darwin" -}}
							d
						{{- else if eq .Os "linux" -}}
							l
						{{- end -}}`,
					},
				},
			},
		},
	})
	build, err := Default.WithDefaults(ctx.Config.Builds[0])
	require.NoError(t, err)
	require.NoError(t, Default.Prepare(ctx, build))

	options := api.Options{
		Name:   "proj",
		Path:   filepath.Join(dist, "proj-aarch64-apple-darwin", "proj"),
		Target: nil,
	}
	options.Target, err = Default.Parse("aarch64-apple-darwin")
	require.NoError(t, err)

	require.NoError(t, Default.Build(ctx, build, options))

	bins := ctx.Artifacts.List()
	require.Len(t, bins, 1)

	bin := bins[0]
	require.Equal(t, artifact.Artifact{
		Name:   "proj",
		Path:   options.Path,
		Goos:   "darwin",
		Goarch: "arm64",
		Target: "aarch64-apple-darwin",
		Type:   artifact.Binary,
		Extra: artifact.Extras{
			artifact.ExtraBinary:  "proj",
			artifact.ExtraBuilder: "rust",
			artifact.ExtraExt:     "",
			artifact.ExtraID:      "default",
		},
	}, *bin)

	require.FileExists(t, bin.Path)
	fi, err := os.Stat(bin.Path)
	require.NoError(t, err)
	require.True(t, modTime.Equal(fi.ModTime()), "inconsistent mod times found when specifying ModTimestamp")
}
