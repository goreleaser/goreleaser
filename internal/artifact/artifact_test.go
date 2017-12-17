package artifact

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

func TestAdd(t *testing.T) {
	var g errgroup.Group
	var artifacts = New()
	for _, a := range []Artifact{
		{
			Name: "foo",
			Type: Archive,
		},
		{
			Name: "bar",
			Type: Binary,
		},
		{
			Name: "foobar",
			Type: DockerImage,
		},
		{
			Name: "check",
			Type: Checksum,
		},
	} {
		a := a
		g.Go(func() error {
			artifacts.Add(a)
			return nil
		})
	}
	assert.NoError(t, g.Wait())
	assert.Len(t, artifacts.items, 4)
}

func TestFilter(t *testing.T) {
	var data = []Artifact{
		{
			Name: "foo",
			Goos: "linux",
		},
		{
			Name:   "bar",
			Goarch: "amd64",
		},
		{
			Name:  "foobar",
			Goarm: "6",
		},
		{
			Name: "check",
			Type: Checksum,
		},
	}
	var artifacts = New()
	for _, a := range data {
		artifacts.Add(a)
	}
	assert.Len(t, artifacts.Filter(ByGoos("linux")).items, 1)
	assert.Len(t, artifacts.Filter(ByGoos("darwin")).items, 0)

	assert.Len(t, artifacts.Filter(ByGoarch("amd64")).items, 1)
	assert.Len(t, artifacts.Filter(ByGoarch("386")).items, 0)

	assert.Len(t, artifacts.Filter(ByGoarm("6")).items, 1)
	assert.Len(t, artifacts.Filter(ByGoarm("7")).items, 0)

	assert.Len(t, artifacts.Filter(ByType(Checksum)).items, 1)
	assert.Len(t, artifacts.Filter(ByType(Binary)).items, 0)
}
