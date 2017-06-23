package context

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

func TestMultipleArtifactAdds(t *testing.T) {
	var assert = assert.New(t)
	var list = []string{
		"dist/a",
		"dist/b",
		"dist/c",
		"dist/d",
	}
	var ctx = New(config.Project{
		Dist: "dist",
	})
	var g errgroup.Group
	for _, f := range list {
		f := f
		g.Go(func() error {
			ctx.AddArtifact(f)
			return nil
		})
	}
	assert.NoError(g.Wait())
	assert.Len(ctx.Artifacts, len(list))
	assert.Contains(ctx.Artifacts, "a", "b", "c", "d")
}
