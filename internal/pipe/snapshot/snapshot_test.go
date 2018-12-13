package snapshot

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/assert"
)

func TestStringer(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}
func TestDefault(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Snapshot: config.Snapshot{},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "SNAPSHOT-{{ .ShortCommit }}", ctx.Config.Snapshot.NameTemplate)
}

func TestDefaultSet(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Snapshot: config.Snapshot{
				NameTemplate: "snap",
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "snap", ctx.Config.Snapshot.NameTemplate)
}

func TestSnapshotNameShortCommitHash(t *testing.T) {
	var ctx = context.New(config.Project{
		Snapshot: config.Snapshot{
			NameTemplate: "{{.ShortCommit}}",
		},
	})
	ctx.Snapshot = true
	ctx.Config.Git.ShortHash = true
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Git.ShortCommit = "123"
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Equal(t, ctx.Version, "123")
}

func TestSnapshotInvalidNametemplate(t *testing.T) {
	var ctx = context.New(config.Project{
		Snapshot: config.Snapshot{
			NameTemplate: "{{.ShortCommit}{{{sss}}}",
		},
	})
	ctx.Snapshot = true
	assert.EqualError(t, Pipe{}.Run(ctx), `failed to generate snapshot name: template: tmpl:1: unexpected "}" in operand`)
}

func TestSnapshotEmptyFinalName(t *testing.T) {
	var ctx = context.New(config.Project{
		Snapshot: config.Snapshot{
			NameTemplate: "{{ .Commit }}",
		},
	})
	ctx.Snapshot = true
	ctx.Git.CurrentTag = "v1.2.3"
	assert.EqualError(t, Pipe{}.Run(ctx), "empty snapshot name")
}

func TestNotASnapshot(t *testing.T) {
	var ctx = context.New(config.Project{})
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}
