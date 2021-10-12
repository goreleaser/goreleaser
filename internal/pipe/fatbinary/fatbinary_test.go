package fatbinary

import (
	"debug/macho"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestDefault(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		ctx := &context.Context{
			Config: config.Project{
				ProjectName: "proj",
				MacOSFatBinaries: []config.FatBinary{
					{},
				},
			},
		}
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, config.FatBinary{
			ID:           "proj",
			NameTemplate: "{{ .ProjectName }}",
		}, ctx.Config.MacOSFatBinaries[0])
	})

	t.Run("given id", func(t *testing.T) {
		ctx := &context.Context{
			Config: config.Project{
				ProjectName: "proj",
				MacOSFatBinaries: []config.FatBinary{
					{ID: "foo"},
				},
			},
		}
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, config.FatBinary{
			ID:           "foo",
			NameTemplate: "{{ .ProjectName }}",
		}, ctx.Config.MacOSFatBinaries[0])
	})

	t.Run("given name", func(t *testing.T) {
		ctx := &context.Context{
			Config: config.Project{
				ProjectName: "proj",
				MacOSFatBinaries: []config.FatBinary{
					{NameTemplate: "foo"},
				},
			},
		}
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, config.FatBinary{
			ID:           "proj",
			NameTemplate: "foo",
		}, ctx.Config.MacOSFatBinaries[0])
	})

	t.Run("duplicated ids", func(t *testing.T) {
		ctx := &context.Context{
			Config: config.Project{
				ProjectName: "proj",
				MacOSFatBinaries: []config.FatBinary{
					{ID: "foo"},
					{ID: "foo"},
				},
			},
		}
		require.EqualError(t, Pipe{}.Default(ctx), `found 2 macos_fatbins with the ID 'foo', please fix your config`)
	})
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			MacOSFatBinaries: []config.FatBinary{{}},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestRun(t *testing.T) {
	dist := t.TempDir()

	src := filepath.Join("testdata", "fake", "main.go")
	paths := map[string]string{
		"amd64": filepath.Join(dist, "fake_darwin_amd64/fake"),
		"arm64": filepath.Join(dist, "fake_darwin_arm64/fake"),
	}

	ctx1 := context.New(config.Project{
		Dist: dist,
		MacOSFatBinaries: []config.FatBinary{
			{
				ID:           "foo",
				NameTemplate: "foo",
				Replace:      true,
			},
		},
	})

	ctx2 := context.New(config.Project{
		Dist: dist,
		MacOSFatBinaries: []config.FatBinary{
			{
				ID:           "foo",
				NameTemplate: "foo",
			},
		},
	})

	ctx3 := context.New(config.Project{
		Dist: dist,
		MacOSFatBinaries: []config.FatBinary{
			{
				ID:           "notfoo",
				NameTemplate: "notfoo",
			},
		},
	})

	ctx4 := context.New(config.Project{
		Dist: dist,
		MacOSFatBinaries: []config.FatBinary{
			{
				ID:           "foo",
				NameTemplate: "foo",
			},
		},
	})

	for arch, path := range paths {
		cmd := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", path, src)
		cmd.Env = append(os.Environ(), "GOOS=darwin", "GOARCH="+arch)
		out, err := cmd.CombinedOutput()
		t.Log(string(out))
		require.NoError(t, err)

		modTime := time.Unix(0, 0)
		require.NoError(t, os.Chtimes(path, modTime, modTime))

		art := artifact.Artifact{
			Name:   "fake",
			Path:   path,
			Goos:   "darwin",
			Goarch: arch,
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				"Binary": "fake",
				"ID":     "foo",
			},
		}
		ctx1.Artifacts.Add(&art)
		ctx2.Artifacts.Add(&art)
		ctx4.Artifacts.Add(&artifact.Artifact{
			Name:   "fake",
			Path:   path + "wrong",
			Goos:   "darwin",
			Goarch: arch,
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				"Binary": "fake",
				"ID":     "foo",
			},
		})
	}

	t.Run("replacing", func(t *testing.T) {
		require.NoError(t, Pipe{}.Run(ctx1))
		require.Len(t, ctx1.Artifacts.Filter(artifact.ByType(artifact.Binary)).List(), 0)
		require.Len(t, ctx1.Artifacts.Filter(artifact.ByType(artifact.FatBinary)).List(), 1)
		checkFatBinary(t, ctx1.Artifacts.Filter(artifact.ByType(artifact.FatBinary)).List()[0])
	})

	t.Run("keeping", func(t *testing.T) {
		require.NoError(t, Pipe{}.Run(ctx2))
		require.Len(t, ctx2.Artifacts.Filter(artifact.ByType(artifact.Binary)).List(), 2)
		require.Len(t, ctx2.Artifacts.Filter(artifact.ByType(artifact.FatBinary)).List(), 1)
		checkFatBinary(t, ctx2.Artifacts.Filter(artifact.ByType(artifact.FatBinary)).List()[0])
	})

	t.Run("bad template", func(t *testing.T) {
		require.EqualError(t, Pipe{}.Run(context.New(config.Project{
			MacOSFatBinaries: []config.FatBinary{
				{
					NameTemplate: "{{.Name}",
				},
			},
		})), `template: tmpl:1: unexpected "}" in operand`)
	})

	t.Run("no darwin builds", func(t *testing.T) {
		require.EqualError(t, Pipe{}.Run(ctx3), `no darwin binaries found with id "notfoo"`)
	})

	t.Run("fail to open", func(t *testing.T) {
		require.ErrorIs(t, Pipe{}.Run(ctx4), os.ErrNotExist)
	})
}

func checkFatBinary(tb testing.TB, fatbin *artifact.Artifact) {
	tb.Helper()

	f, err := macho.OpenFat(fatbin.Path)
	require.NoError(tb, err)
	require.Len(tb, f.Arches, 2)

	check, err := fatbin.Checksum("sha256")
	require.NoError(tb, err)
	require.Equal(tb, "53692e6fbb45ce90c6da6a178fad501829e9f14d82c62435c3a3c7ed8a04e0bb", check)
}
