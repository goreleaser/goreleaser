package snapshot

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestDefault(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Snapshot: config.Snapshot{},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "{{ .Version }}-SNAPSHOT-{{ .ShortCommit }}", ctx.Config.Snapshot.NameTemplate)
}

func TestDefaultSet(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Snapshot: config.Snapshot{
				NameTemplate: "snap",
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "snap", ctx.Config.Snapshot.NameTemplate)
}

func TestSnapshotInvalidNametemplate(t *testing.T) {
	ctx := context.New(config.Project{
		Snapshot: config.Snapshot{
			NameTemplate: "{{.ShortCommit}{{{sss}}}",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), `failed to generate snapshot name: template: tmpl:1: unexpected "}" in operand`)
}

func TestSnapshotEmptyFinalName(t *testing.T) {
	ctx := context.New(config.Project{
		Snapshot: config.Snapshot{
			NameTemplate: "{{ .Commit }}",
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	require.EqualError(t, Pipe{}.Run(ctx), "empty snapshot name")
}

func TestSnapshot(t *testing.T) {
	ctx := context.New(config.Project{
		Snapshot: config.Snapshot{
			NameTemplate: "{{ incpatch .Tag }}",
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "v1.2.4", ctx.Version)
}

func TestSkip(t *testing.T) {
	ctx := context.New(config.Project{})
	require.True(t, Pipe{}.Skip(ctx))
}
