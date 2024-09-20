package snapshot

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestDefault(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Snapshot: config.Snapshot{},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "{{ .Version }}-SNAPSHOT-{{ .ShortCommit }}", ctx.Config.Snapshot.VersionTemplate)
}

func TestDefaultDeprecated(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Snapshot: config.Snapshot{
			NameTemplate: "snap",
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "snap", ctx.Config.Snapshot.VersionTemplate)
}

func TestDefaultSet(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Snapshot: config.Snapshot{
			VersionTemplate: "snap",
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "snap", ctx.Config.Snapshot.VersionTemplate)
}

func TestSnapshotInvalidVersionTemplate(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Snapshot: config.Snapshot{
			VersionTemplate: "{{.ShortCommit}{{{sss}}}",
		},
	})
	testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
}

func TestSnapshotEmptyFinalName(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Snapshot: config.Snapshot{
			VersionTemplate: "{{ .Commit }}",
		},
	}, testctx.WithCurrentTag("v1.2.3"))
	require.EqualError(t, Pipe{}.Run(ctx), "empty snapshot name")
}

func TestSnapshot(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Snapshot: config.Snapshot{
			VersionTemplate: "{{ incpatch .Version }}",
		},
	}, testctx.WithVersion("1.2.3"))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "1.2.4", ctx.Version)
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.New()))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.New(testctx.Snapshot)
		require.False(t, Pipe{}.Skip(ctx))
	})
}
