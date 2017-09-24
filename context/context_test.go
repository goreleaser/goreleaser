package context

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

func TestMultipleAdds(t *testing.T) {
	var assert = assert.New(t)
	var artifacts = []string{
		"dist/a",
		"dist/b",
		"dist/c",
		"dist/d",
	}
	var dockerfiles = []string{
		"a/b:1.0.0",
		"c/d:2.0.0",
		"e/f:3.0.0",
	}
	var ctx = New(config.Project{
		Dist: "dist",
	})
	var g errgroup.Group
	for _, f := range artifacts {
		f := f
		g.Go(func() error {
			ctx.AddArtifact(f)
			return nil
		})
	}
	assert.NoError(g.Wait())
	for _, d := range dockerfiles {
		d := d
		g.Go(func() error {
			ctx.AddDocker(d)
			return nil
		})
	}
	assert.NoError(g.Wait())
	assert.Len(ctx.Artifacts, len(artifacts))
	assert.Contains(ctx.Artifacts, "a", "b", "c", "d")
	assert.Len(ctx.Dockers, len(dockerfiles))
	assert.Contains(ctx.Dockers, "a/b:1.0.0", "c/d:2.0.0", "e/f:3.0.0")
}

func TestMultipleBinaryAdds(t *testing.T) {
	var assert = assert.New(t)
	var list = map[string]string{
		"a": "folder/a",
		"b": "folder/b",
		"c": "folder/c",
		"d": "folder/d",
	}
	var ctx = New(config.Project{
		Dist: "dist",
	})
	var g errgroup.Group
	for k, f := range list {
		f := f
		k := k
		g.Go(func() error {
			ctx.AddBinary("linuxamd64", k, k, f)
			return nil
		})
	}
	assert.NoError(g.Wait())
	assert.Len(ctx.Binaries["linuxamd64"], len(list))
	assert.Len(ctx.Binaries, 1)
}
