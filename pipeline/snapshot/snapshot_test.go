package snapshot

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
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
	assert.Equal(t, "SNAPSHOT-{{ .Commit }}", ctx.Config.Snapshot.NameTemplate)
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
