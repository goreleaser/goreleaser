package artifact

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// ensure Type implements the stringer interface...
var _ fmt.Stringer = Type(0)

func TestAdd(t *testing.T) {
	var g errgroup.Group
	artifacts := New()
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
	require.NoError(t, g.Wait())
	require.Len(t, artifacts.List(), 4)
}

func TestFilter(t *testing.T) {
	data := []*Artifact{
		{
			Name:   "foo",
			Goos:   "linux",
			Goarch: "arm",
		},
		{
			Name:    "bar",
			Goarch:  "amd64",
			Goamd64: "v1",
		},
		{
			Name:    "bar",
			Goarch:  "amd64",
			Goamd64: "v2",
		},
		{
			Name:    "bar",
			Goarch:  "amd64",
			Goamd64: "v3",
		},
		{
			Name:    "bar",
			Goarch:  "amd64",
			Goamd64: "v4",
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
		{
			Name:   "unibin-replaces",
			Goos:   "darwin",
			Goarch: "all",
			Extra: map[string]interface{}{
				ExtraReplaces: true,
			},
		},
		{
			Name:   "unibin-noreplace",
			Goos:   "darwin",
			Goarch: "all",
			Extra: map[string]interface{}{
				ExtraReplaces: false,
			},
		},
	}
	artifacts := New()
	for _, a := range data {
		artifacts.Add(a)
	}

	require.Len(t, artifacts.Filter(ByGoos("linux")).items, 1)
	require.Len(t, artifacts.Filter(ByGoos("darwin")).items, 2)

	require.Len(t, artifacts.Filter(ByGoarch("amd64")).items, 4)
	require.Len(t, artifacts.Filter(ByGoarch("386")).items, 0)

	require.Len(t, artifacts.Filter(ByGoamd64("v1")).items, 1)
	require.Len(t, artifacts.Filter(ByGoamd64("v2")).items, 1)
	require.Len(t, artifacts.Filter(ByGoamd64("v3")).items, 1)
	require.Len(t, artifacts.Filter(ByGoamd64("v4")).items, 1)

	require.Len(t, artifacts.Filter(ByGoarm("6")).items, 1)
	require.Len(t, artifacts.Filter(ByGoarm("7")).items, 0)

	require.Len(t, artifacts.Filter(ByType(Checksum)).items, 2)
	require.Len(t, artifacts.Filter(ByType(Binary)).items, 0)

	require.Len(t, artifacts.Filter(OnlyReplacingUnibins).items, 9)
	require.Len(t, artifacts.Filter(And(OnlyReplacingUnibins, ByGoos("darwin"))).items, 1)

	require.Len(t, artifacts.Filter(nil).items, 10)

	require.Len(t, artifacts.Filter(
		And(
			ByType(Checksum),
			func(a *Artifact) bool {
				return a.Name == "checkzumm"
			},
		),
	).List(), 1)

	require.Len(t, artifacts.Filter(
		Or(
			ByType(Checksum),
			And(
				ByGoos("linux"),
				ByGoarm("arm"),
			),
		),
	).List(), 2)
}

func TestRemove(t *testing.T) {
	data := []*Artifact{
		{
			Name:   "foo",
			Goos:   "linux",
			Goarch: "arm",
			Type:   Binary,
		},
		{
			Name:   "universal",
			Goos:   "darwin",
			Goarch: "all",
			Type:   UniversalBinary,
		},
		{
			Name:   "bar",
			Goarch: "amd64",
		},
		{
			Name: "checks",
			Type: Checksum,
		},
	}

	t.Run("null filter", func(t *testing.T) {
		artifacts := New()
		for _, a := range data {
			artifacts.Add(a)
		}
		require.NoError(t, artifacts.Remove(nil))
		require.Len(t, artifacts.List(), len(data))
	})

	t.Run("removing", func(t *testing.T) {
		artifacts := New()
		for _, a := range data {
			artifacts.Add(a)
		}
		require.NoError(t, artifacts.Remove(
			Or(
				ByType(Checksum),
				ByType(UniversalBinary),
				And(
					ByGoos("linux"),
					ByGoarch("arm"),
				),
			),
		))

		require.Len(t, artifacts.List(), 1)
	})
}

func TestGroupByID(t *testing.T) {
	data := []*Artifact{
		{
			Name: "foo",
			Extra: map[string]interface{}{
				ExtraID: "foo",
			},
		},
		{
			Name: "bar",
			Extra: map[string]interface{}{
				ExtraID: "foo",
			},
		},
		{
			Name: "foobar",
			Goos: "linux",
			Extra: map[string]interface{}{
				ExtraID: "foovar",
			},
		},
		{
			Name: "foobar",
			Goos: "linux",
			Extra: map[string]interface{}{
				ExtraID: "foovar",
			},
		},
		{
			Name: "foobar",
			Goos: "linux",
			Extra: map[string]interface{}{
				ExtraID: "foobar",
			},
		},
		{
			Name: "check",
			Type: Checksum,
		},
	}
	artifacts := New()
	for _, a := range data {
		artifacts.Add(a)
	}

	groups := artifacts.GroupByID()
	require.Len(t, groups["foo"], 2)
	require.Len(t, groups["foobar"], 1)
	require.Len(t, groups["foovar"], 2)
	require.Len(t, groups, 3)
}

func TestGroupByPlatform(t *testing.T) {
	data := []*Artifact{
		{
			Name:    "foo",
			Goos:    "linux",
			Goarch:  "amd64",
			Goamd64: "v2",
		},
		{
			Name:    "bar",
			Goos:    "linux",
			Goarch:  "amd64",
			Goamd64: "v2",
		},
		{
			Name:    "bar",
			Goos:    "linux",
			Goarch:  "amd64",
			Goamd64: "v3",
		},
		{
			Name:   "foobar",
			Goos:   "linux",
			Goarch: "arm",
			Goarm:  "6",
		},
		{
			Name:   "foobar",
			Goos:   "linux",
			Goarch: "mips",
			Goarm:  "softfloat",
		},
		{
			Name:   "foobar",
			Goos:   "linux",
			Goarch: "mips",
			Goarm:  "hardfloat",
		},
		{
			Name: "check",
			Type: Checksum,
		},
	}
	artifacts := New()
	for _, a := range data {
		artifacts.Add(a)
	}

	groups := artifacts.GroupByPlatform()
	require.Len(t, groups["linuxamd64v2"], 2)
	require.Len(t, groups["linuxamd64v3"], 1)
	require.Len(t, groups["linuxarm6"], 1)
	require.Len(t, groups["linuxmipssoftfloat"], 1)
	require.Len(t, groups["linuxmipshardfloat"], 1)
}

func TestChecksum(t *testing.T) {
	folder := t.TempDir()
	file := filepath.Join(folder, "subject")
	require.NoError(t, os.WriteFile(file, []byte("lorem ipsum"), 0o644))

	artifact := Artifact{
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
	file := filepath.Join(t.TempDir(), "nope")
	artifact := Artifact{
		Path: file,
	}
	sum, err := artifact.Checksum("sha1")
	require.ErrorIs(t, err, os.ErrNotExist)
	require.Empty(t, sum)
}

func TestInvalidAlgorithm(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	artifact := Artifact{
		Path: f.Name(),
	}
	sum, err := artifact.Checksum("sha1ssss")
	require.EqualError(t, err, `invalid algorithm: sha1ssss`)
	require.Empty(t, sum)
}

func TestExtra(t *testing.T) {
	a := Artifact{
		Extra: map[string]interface{}{
			"Foo": "foo",
			"docker": config.Docker{
				ID:  "id",
				Use: "docker",
			},
			"fail-plz": config.Homebrew{
				Plist: "aaaa",
			},
			"unsupported": func() {},
			"binaries":    []string{"foo", "bar"},
		},
	}

	t.Run("string", func(t *testing.T) {
		foo, err := Extra[string](a, "Foo")
		require.NoError(t, err)
		require.Equal(t, "foo", foo)
		require.Equal(t, "foo", ExtraOr(a, "Foo", "bar"))
	})

	t.Run("missing field", func(t *testing.T) {
		bar, err := Extra[string](a, "Foobar")
		require.NoError(t, err)
		require.Equal(t, "", bar)
		require.Equal(t, "bar", ExtraOr(a, "Foobar", "bar"))
	})

	t.Run("complex", func(t *testing.T) {
		docker, err := Extra[config.Docker](a, "docker")
		require.NoError(t, err)
		require.Equal(t, "id", docker.ID)
	})

	t.Run("array", func(t *testing.T) {
		binaries, err := Extra[[]string](a, "binaries")
		require.NoError(t, err)
		require.Equal(t, []string{"foo", "bar"}, binaries)
		require.Equal(t, []string{"foo", "bar"}, ExtraOr(a, "binaries", []string{}))
	})

	t.Run("unmarshal error", func(t *testing.T) {
		_, err := Extra[config.Docker](a, "fail-plz")
		require.EqualError(t, err, "json: unknown field \"tap\"")
	})

	t.Run("marshal error", func(t *testing.T) {
		_, err := Extra[config.Docker](a, "unsupported")
		require.EqualError(t, err, "json: unsupported type: func()")
	})
}

func TestByIDs(t *testing.T) {
	data := []*Artifact{
		{
			Name: "foo",
			Extra: map[string]interface{}{
				ExtraID: "foo",
			},
		},
		{
			Name: "bar",
			Extra: map[string]interface{}{
				ExtraID: "bar",
			},
		},
		{
			Name: "foobar",
			Extra: map[string]interface{}{
				ExtraID: "foo",
			},
		},
		{
			Name: "check",
			Extra: map[string]interface{}{
				ExtraID: "check",
			},
		},
		{
			Name: "checksum",
			Type: Checksum,
		},
	}
	artifacts := New()
	for _, a := range data {
		artifacts.Add(a)
	}

	require.Len(t, artifacts.Filter(ByIDs("check")).items, 2)
	require.Len(t, artifacts.Filter(ByIDs("foo")).items, 3)
	require.Len(t, artifacts.Filter(ByIDs("foo", "bar")).items, 4)
}

func TestByExts(t *testing.T) {
	data := []*Artifact{
		{
			Name: "foo",
			Extra: map[string]interface{}{
				ExtraExt: "deb",
			},
		},
		{
			Name: "bar",
			Extra: map[string]interface{}{
				ExtraExt: "deb",
			},
		},
		{
			Name: "foobar",
			Extra: map[string]interface{}{
				ExtraExt: "rpm",
			},
		},
		{
			Name:  "check",
			Extra: map[string]interface{}{},
		},
	}
	artifacts := New()
	for _, a := range data {
		artifacts.Add(a)
	}

	require.Len(t, artifacts.Filter(ByExt("deb")).items, 2)
	require.Len(t, artifacts.Filter(ByExt("rpm")).items, 1)
	require.Len(t, artifacts.Filter(ByExt("rpm", "deb")).items, 3)
	require.Len(t, artifacts.Filter(ByExt("foo")).items, 0)
}

func TestByFormats(t *testing.T) {
	data := []*Artifact{
		{
			Name: "foo",
			Extra: map[string]interface{}{
				ExtraFormat: "zip",
			},
		},
		{
			Name: "bar",
			Extra: map[string]interface{}{
				ExtraFormat: "tar.gz",
			},
		},
		{
			Name: "foobar",
			Extra: map[string]interface{}{
				ExtraFormat: "zip",
			},
		},
		{
			Name: "bin",
			Extra: map[string]interface{}{
				ExtraFormat: "binary",
			},
		},
	}
	artifacts := New()
	for _, a := range data {
		artifacts.Add(a)
	}

	require.Len(t, artifacts.Filter(ByFormats("binary")).items, 1)
	require.Len(t, artifacts.Filter(ByFormats("zip")).items, 2)
	require.Len(t, artifacts.Filter(ByFormats("zip", "tar.gz")).items, 3)
}

func TestPaths(t *testing.T) {
	paths := []string{"a/b", "b/c", "d/e", "f/g"}
	artifacts := New()
	for _, a := range paths {
		artifacts.Add(&Artifact{
			Path: a,
		})
	}
	require.ElementsMatch(t, paths, artifacts.Paths())
}

func TestRefresher(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		artifacts := New()
		path := filepath.Join(t.TempDir(), "f")
		artifacts.Add(&Artifact{
			Name: "f",
			Path: path,
			Type: Checksum,
			Extra: map[string]interface{}{
				"Refresh": func() error {
					return os.WriteFile(path, []byte("hello"), 0o765)
				},
			},
		})
		artifacts.Add(&Artifact{
			Name: "no refresh",
			Type: Checksum,
		})

		for _, item := range artifacts.List() {
			require.NoError(t, item.Refresh())
		}

		bts, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "hello", string(bts))
	})

	t.Run("nok", func(t *testing.T) {
		artifacts := New()
		artifacts.Add(&Artifact{
			Name: "fail",
			Type: Checksum,
			Extra: map[string]interface{}{
				"ID": "nok",
				"Refresh": func() error {
					return fmt.Errorf("fake err")
				},
			},
		})

		for _, item := range artifacts.List() {
			require.EqualError(t, item.Refresh(), `failed to refresh "fail": fake err`)
		}
	})

	t.Run("not a checksum", func(t *testing.T) {
		artifacts := New()
		artifacts.Add(&Artifact{
			Name: "will be ignored",
			Type: Binary,
			Extra: map[string]interface{}{
				"ID": "ignored",
				"Refresh": func() error {
					return fmt.Errorf("err that should not happen")
				},
			},
		})

		for _, item := range artifacts.List() {
			require.NoError(t, item.Refresh())
		}
	})
}

func TestVisit(t *testing.T) {
	artifacts := New()
	artifacts.Add(&Artifact{
		Name: "foo",
		Type: Checksum,
	})
	artifacts.Add(&Artifact{
		Name: "foo",
		Type: Binary,
	})

	t.Run("ok", func(t *testing.T) {
		require.NoError(t, artifacts.Visit(func(a *Artifact) error {
			require.Equal(t, "foo", a.Name)
			return nil
		}))
	})

	t.Run("nok", func(t *testing.T) {
		require.EqualError(t, artifacts.Visit(func(a *Artifact) error {
			return fmt.Errorf("fake err")
		}), `fake err`)
	})
}

func TestMarshalJSON(t *testing.T) {
	artifacts := New()
	artifacts.Add(&Artifact{
		Name: "foo",
		Type: Binary,
		Extra: map[string]interface{}{
			ExtraID: "adsad",
		},
	})
	artifacts.Add(&Artifact{
		Name: "foo",
		Type: UploadableArchive,
		Extra: map[string]interface{}{
			ExtraID: "adsad",
		},
	})
	artifacts.Add(&Artifact{
		Name: "foo",
		Type: Checksum,
		Extra: map[string]interface{}{
			ExtraRefresh: func() error { return nil },
		},
	})
	bts, err := json.Marshal(artifacts.List())
	require.NoError(t, err)
	golden.RequireEqualJSON(t, bts)
}

func Test_ByBinaryLikeArtifacts(t *testing.T) {
	tests := []struct {
		name     string
		initial  []*Artifact
		expected []*Artifact
	}{
		{
			name: "keep all unique paths",
			initial: []*Artifact{
				{
					Path: "binary-path",
					Type: Binary,
				},
				{
					Path: "uploadable-binary-path",
					Type: UploadableBinary,
				},
				{
					Path: "universal-binary-path",
					Type: UniversalBinary,
				},
			},
			expected: []*Artifact{
				{
					Path: "binary-path",
					Type: Binary,
				},
				{
					Path: "uploadable-binary-path",
					Type: UploadableBinary,
				},
				{
					Path: "universal-binary-path",
					Type: UniversalBinary,
				},
			},
		},
		{
			name: "duplicate path between binaries ignored (odd configuration)",
			initial: []*Artifact{
				{
					Path: "!!!duplicate!!!",
					Type: Binary,
				},
				{
					Path: "uploadable-binary-path",
					Type: UploadableBinary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: UniversalBinary,
				},
			},
			expected: []*Artifact{
				{
					Path: "!!!duplicate!!!",
					Type: Binary,
				},
				{
					Path: "uploadable-binary-path",
					Type: UploadableBinary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: UniversalBinary,
				},
			},
		},
		{
			name: "remove duplicate binary",
			initial: []*Artifact{
				{
					Path: "!!!duplicate!!!",
					Type: Binary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: UploadableBinary,
				},
				{
					Path: "universal-binary-path",
					Type: UniversalBinary,
				},
			},
			expected: []*Artifact{
				{
					Path: "!!!duplicate!!!",
					Type: UploadableBinary,
				},
				{
					Path: "universal-binary-path",
					Type: UniversalBinary,
				},
			},
		},
		{
			name: "remove duplicate universal binary",
			initial: []*Artifact{
				{
					Path: "binary-path",
					Type: Binary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: UploadableBinary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: UniversalBinary,
				},
			},
			expected: []*Artifact{
				{
					Path: "binary-path",
					Type: Binary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: UploadableBinary,
				},
			},
		},
		{
			name: "remove multiple duplicates",
			initial: []*Artifact{
				{
					Path: "!!!duplicate!!!",
					Type: Binary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: Binary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: UploadableBinary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: UniversalBinary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: UniversalBinary,
				},
			},
			expected: []*Artifact{
				{
					Path: "!!!duplicate!!!",
					Type: UploadableBinary,
				},
			},
		},
		{
			name: "keep duplicate uploadable binaries (odd configuration)",
			initial: []*Artifact{
				{
					Path: "!!!duplicate!!!",
					Type: Binary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: Binary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: UploadableBinary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: UploadableBinary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: UniversalBinary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: UniversalBinary,
				},
			},
			expected: []*Artifact{
				{
					Path: "!!!duplicate!!!",
					Type: UploadableBinary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: UploadableBinary,
				},
			},
		},
		{
			name: "keeps duplicates when there is no uploadable binary",
			initial: []*Artifact{
				{
					Path: "!!!duplicate!!!",
					Type: Binary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: Binary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: UniversalBinary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: UniversalBinary,
				},
			},
			expected: []*Artifact{
				{
					Path: "!!!duplicate!!!",
					Type: Binary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: Binary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: UniversalBinary,
				},
				{
					Path: "!!!duplicate!!!",
					Type: UniversalBinary,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arts := New()
			for _, a := range tt.initial {
				arts.Add(a)
			}
			actual := arts.Filter(ByBinaryLikeArtifacts(arts)).List()
			expected := New()
			for _, a := range tt.expected {
				expected.Add(a)
			}
			assert.Equal(t, expected.List(), actual)

			if t.Failed() {
				t.Log("expected:")
				for _, a := range tt.expected {
					t.Logf("   %s: %s", a.Type.String(), a.Path)
				}

				t.Log("got:")
				for _, a := range actual {
					t.Logf("   %s: %s", a.Type.String(), a.Path)
				}
			}
		})
	}
}

func TestArtifactStringer(t *testing.T) {
	require.Equal(t, "foobar", Artifact{
		Name: "foobar",
	}.String())
}
