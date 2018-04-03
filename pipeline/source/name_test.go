package source

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestNameFor(t *testing.T) {
	ctx := context.New(config.Project{
		ProjectName: "mybin",
		Builds: []config.Build{
			{
				Binary: "mybin",
			},
		},
		Source: config.Source{
			NameTemplate: "{{.Binary}}-{{.Version}}",
		},
	})
	ctx.Version = "testversion"
	ctx.Git.Commit = "FEFEFEFE"

	name, err := nameFor(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "mybin-testversion", name)
}
