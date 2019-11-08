package artifact

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// ensure Type implements the stringer interface...
var _ fmt.Stringer = Type(0)

func TestAdd(t *testing.T) {
	var g errgroup.Group
	var artifacts = New()
	for _, a := range []*Artifact{
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
	var data = []*Artifact{
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
			func(a *Artifact) bool {
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
	var data = []*Artifact{
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

func TestChecksum(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	require.NoError(t, err)
	var file = filepath.Join(folder, "subject")
	require.NoError(t, ioutil.WriteFile(file, []byte("lorem ipsum"), 0644))

	var artifact = Artifact{
		Path: file,
	}

	for algo, result := range map[string]string{
		"sha256": "5e2bf57d3f40c4b6df69daf1936cb766f832374b4fc0259a7cbff06e2f70f269",
		"sha512": "f80eebd9aabb1a15fb869ed568d858a5c0dca3d5da07a410e1bd988763918d973e344814625f7c844695b2de36ffd27af290d0e34362c51dee5947d58d40527a",
		"sha1":   "bfb7759a67daeb65410490b4d98bb9da7d1ea2ce",
		"crc32":  "72d7748e",
		"md5":    "80a751fde577028640c419000e33eba6",
		"sha224": "e191edf06005712583518ced92cc2ac2fac8d6e4623b021a50736a91",
		"sha384": "597493a6cf1289757524e54dfd6f68b332c7214a716a3358911ef5c09907adc8a654a18c1d721e183b0025f996f6e246",
	} {
		t.Run(algo, func(t *testing.T) {
			sum, err := artifact.Checksum(algo)
			require.NoError(t, err)
			require.Equal(t, result, sum)
		})
	}
}

func TestChecksumFileDoesntExist(t *testing.T) {
	var artifact = Artifact{
		Path: "/tmp/adasdasdas/asdasd/asdas",
	}
	sum, err := artifact.Checksum("sha1")
	require.EqualError(t, err, `failed to checksum: open /tmp/adasdasdas/asdasd/asdas: no such file or directory`)
	require.Empty(t, sum)
}

func TestInvalidAlgorithm(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	var artifact = Artifact{
		Path: f.Name(),
	}
	sum, err := artifact.Checksum("sha1ssss")
	require.EqualError(t, err, `invalid algorith: sha1ssss`)
	require.Empty(t, sum)
}

func TestExtraOr(t *testing.T) {
	var a = &Artifact{
		Extra: map[string]interface{}{
			"Foo": "foo",
		},
	}
	require.Equal(t, "foo", a.ExtraOr("Foo", "bar"))
	require.Equal(t, "bar", a.ExtraOr("Foobar", "bar"))
}

func TestByIDs(t *testing.T) {
	var data = []*Artifact{
		{
			Name: "foo",
			Extra: map[string]interface{}{
				"ID": "foo",
			},
		},
		{
			Name: "bar",
			Extra: map[string]interface{}{
				"ID": "bar",
			},
		},
		{
			Name: "foobar",
			Extra: map[string]interface{}{
				"ID": "foo",
			},
		},
		{
			Name: "check",
			Extra: map[string]interface{}{
				"ID": "check",
			},
		},
		{
			Name: "checksum",
			Type: Checksum,
		},
	}
	var artifacts = New()
	for _, a := range data {
		artifacts.Add(a)
	}

	require.Len(t, artifacts.Filter(ByIDs("check")).items, 2)
	require.Len(t, artifacts.Filter(ByIDs("foo")).items, 3)
	require.Len(t, artifacts.Filter(ByIDs("foo", "bar")).items, 4)
}

func TestByFormats(t *testing.T) {
	var data = []*Artifact{
		{
			Name: "foo",
			Extra: map[string]interface{}{
				"Format": "zip",
			},
		},
		{
			Name: "bar",
			Extra: map[string]interface{}{
				"Format": "tar.gz",
			},
		},
		{
			Name: "foobar",
			Extra: map[string]interface{}{
				"Format": "zip",
			},
		},
		{
			Name: "bin",
			Extra: map[string]interface{}{
				"Format": "binary",
			},
		},
	}
	var artifacts = New()
	for _, a := range data {
		artifacts.Add(a)
	}

	require.Len(t, artifacts.Filter(ByFormats("binary")).items, 1)
	require.Len(t, artifacts.Filter(ByFormats("zip")).items, 2)
	require.Len(t, artifacts.Filter(ByFormats("zip", "tar.gz")).items, 3)
}
