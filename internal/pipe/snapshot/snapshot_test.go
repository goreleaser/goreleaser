package snapshot

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
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
	require.Equal(t, "{{ .Version }}-SNAPSHOT-{{ .ShortCommit }}", ctx.Config.Snapshot.NameTemplate)
}

func TestDefaultSet(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Snapshot: config.Snapshot{
			NameTemplate: "snap",
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "snap", ctx.Config.Snapshot.NameTemplate)
}

func TestSnapshotInvalidNametemplate(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Snapshot: config.Snapshot{
			NameTemplate: "{{.ShortCommit}{{{sss}}}",
		},
	})
	testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
}

func TestSnapshotEmptyFinalName(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Snapshot: config.Snapshot{
			NameTemplate: "{{ .Commit }}",
		},
	}, testctx.WithCurrentTag("v1.2.3"))
	require.EqualError(t, Pipe{}.Run(ctx), "empty snapshot name")
}

func TestSnapshot(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Snapshot: config.Snapshot{
			NameTemplate: "{{ incpatch .Tag }}",
		},
	}, testctx.WithCurrentTag("v1.2.3"))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "v1.2.4", ctx.Version)
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
