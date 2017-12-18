package artifact

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

// ensure Type implements the stringer interface...
var _ fmt.Stringer = Type(0)

func TestAdd(t *testing.T) {
	var g errgroup.Group
	var artifacts = New()
	for _, a := range []Artifact{
		{
			Name: "foo",
			Type: UploadableArchive,
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
	assert.Len(t, artifacts.List(), 4)
}

func TestFilter(t *testing.T) {
	var data = []Artifact{
		{
			Name:   "foo",
			Goos:   "linux",
			Goarch: "arm",
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
		{
			Name: "checkzumm",
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

	assert.Len(t, artifacts.Filter(ByType(Checksum)).items, 2)
	assert.Len(t, artifacts.Filter(ByType(Binary)).items, 0)

	assert.Len(t, artifacts.Filter(
		And(
			ByType(Checksum),
			func(a Artifact) bool {
				return a.Name == "checkzumm"
			},
		),
	).List(), 1)

	assert.Len(t, artifacts.Filter(
		Or(
			ByType(Checksum),
			And(
				ByGoos("linux"),
				ByGoarm("arm"),
			),
		),
	).List(), 2)
}

func TestGroupByPlatform(t *testing.T) {
	var data = []Artifact{
		{
			Name:   "foo",
			Goos:   "linux",
			Goarch: "amd64",
		},
		{
			Name:   "bar",
			Goos:   "linux",
			Goarch: "amd64",
		},
		{
			Name:   "foobar",
			Goos:   "linux",
			Goarch: "arm",
			Goarm:  "6",
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

	var groups = artifacts.GroupByPlatform()
	assert.Len(t, groups["linuxamd64"], 2)
	assert.Len(t, groups["linuxarm6"], 1)
}
