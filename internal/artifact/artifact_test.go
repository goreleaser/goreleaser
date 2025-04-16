package artifact

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/golden"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// ensure Type implements the stringer interface...
var _ fmt.Stringer = Type(0)

func TestAdd(t *testing.T) {
	var g errgroup.Group
	artifacts := New()
	wd, _ := os.Getwd()
	for _, a := range []*Artifact{
		{
			Name: " whitespaces .zip",
			Type: UploadableArchive,
			Path: filepath.Join(wd, "/foo/bar.tgz"),
		},
		{
			Name: "bar",
			Type: Binary,
		},
		{
			Name: " whitespaces ",
			Type: UploadableBinary,
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
		g.Go(func() error {
			artifacts.Add(a)
			return nil
		})
	}
	require.NoError(t, g.Wait())
	require.Len(t, artifacts.List(), 5)
	archives := artifacts.Filter(ByType(UploadableArchive)).List()
	require.Len(t, archives, 1)
	require.Equal(t, "whitespaces.zip", archives[0].Name)
	binaries := artifacts.Filter(ByType(UploadableBinary)).List()
	require.Len(t, binaries, 1)
	require.Equal(t, "whitespaces", binaries[0].Name)
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
			Name:   "bar",
			Goarch: "amd64",
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
			Name:   "foobar",
			Goarch: "arm",
			Goarm:  "6",
		},
		{
			Name:   "foobar",
			Goarch: "arm",
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
			Extra: map[string]any{
				ExtraReplaces: true,
			},
		},
		{
			Name:   "unibin-noreplace",
			Goos:   "darwin",
			Goarch: "all",
			Extra: map[string]any{
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

	require.Len(t, artifacts.Filter(ByGoarch("amd64")).items, 5)
	require.Empty(t, artifacts.Filter(ByGoarch("386")).items)

	require.Len(t, artifacts.Filter(And(ByGoarch("amd64"), ByGoamd64("v1"))).items, 2)
	require.Len(t, artifacts.Filter(ByGoamd64("v2")).items, 1)
	require.Len(t, artifacts.Filter(ByGoamd64("v3")).items, 1)
	require.Len(t, artifacts.Filter(ByGoamd64("v4")).items, 1)

	require.Len(t, artifacts.Filter(And(ByGoarch("arm"), ByGoarm("6"))).items, 3)
	require.Empty(t, artifacts.Filter(ByGoarm("7")).items)

	require.Len(t, artifacts.Filter(ByType(Checksum)).items, 2)
	require.Empty(t, artifacts.Filter(ByType(Binary)).items)

	require.Len(t, artifacts.Filter(OnlyReplacingUnibins).items, 11)
	require.Len(t, artifacts.Filter(And(OnlyReplacingUnibins, ByGoos("darwin"))).items, 1)

	require.Len(t, artifacts.Filter(nil).items, 12)

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
			Extra: map[string]any{
				ExtraID: "foo",
			},
		},
		{
			Name: "bar",
			Extra: map[string]any{
				ExtraID: "foo",
			},
		},
		{
			Name: "foobar",
			Goos: "linux",
			Extra: map[string]any{
				ExtraID: "foovar",
			},
		},
		{
			Name: "foobar",
			Goos: "linux",
			Extra: map[string]any{
				ExtraID: "foovar",
			},
		},
		{
			Name: "foobar",
			Goos: "linux",
			Extra: map[string]any{
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
			Gomips: "softfloat",
		},
		{
			Name:   "foobar",
			Goos:   "linux",
			Goarch: "mips",
			Gomips: "hardfloat",
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

func TestGroupByPlatform_mixingBuilders(t *testing.T) {
	data := []*Artifact{
		{
			Name:   "foo",
			Goos:   "linux",
			Goarch: "amd64",
		},
		{
			Name:    "bar",
			Goos:    "linux",
			Goarch:  "amd64",
			Goamd64: "v1",
		},
		{
			Name:   "foo",
			Goos:   "linux",
			Goarch: "mips",
		},
		{
			Name:   "bar",
			Goos:   "linux",
			Goarch: "mips",
			Gomips: "hardfloat",
		},
		{
			Name:   "foo",
			Goos:   "linux",
			Goarch: "arm",
			Goarm:  "6",
		},
		{
			Name:   "bar",
			Goos:   "linux",
			Goarch: "arm",
		},
	}
	artifacts := New()
	for _, a := range data {
		artifacts.Add(a)
	}
	groups := artifacts.GroupByPlatform()
	require.Len(t, groups, 3)
	require.Len(t, groups["linuxamd64"], 2)
	require.Len(t, groups["linuxmips"], 2)
	require.Len(t, groups["linuxarm"], 2)
}

func TestGroupByPlatform_abi(t *testing.T) {
	data := []*Artifact{
		{
			Name:   "foo",
			Goos:   "linux",
			Goarch: "amd64",
			Extra: map[string]any{
				"Abi": "musl",
			},
		},
		{
			Name:   "foo",
			Goos:   "linux",
			Goarch: "amd64",
			Extra: map[string]any{
				"Abi": "gnu",
			},
		},
	}
	artifacts := New()
	for _, a := range data {
		artifacts.Add(a)
	}
	groups := artifacts.GroupByPlatform()
	require.Len(t, groups, 2)
	require.Len(t, groups["linuxamd64musl"], 1)
	require.Len(t, groups["linuxamd64gnu"], 1)
}

func TestChecksum(t *testing.T) {
	folder := t.TempDir()
	file := filepath.Join(folder, "subject")
	require.NoError(t, os.WriteFile(file, []byte("lorem ipsum"), 0o644))

	artifact := Artifact{
		Path: file,
	}

	for algo, result := range map[string]string{
		"sha256":   "5e2bf57d3f40c4b6df69daf1936cb766f832374b4fc0259a7cbff06e2f70f269",
		"sha512":   "f80eebd9aabb1a15fb869ed568d858a5c0dca3d5da07a410e1bd988763918d973e344814625f7c844695b2de36ffd27af290d0e34362c51dee5947d58d40527a",
		"sha1":     "bfb7759a67daeb65410490b4d98bb9da7d1ea2ce",
		"crc32":    "72d7748e",
		"md5":      "80a751fde577028640c419000e33eba6",
		"sha224":   "e191edf06005712583518ced92cc2ac2fac8d6e4623b021a50736a91",
		"sha384":   "597493a6cf1289757524e54dfd6f68b332c7214a716a3358911ef5c09907adc8a654a18c1d721e183b0025f996f6e246",
		"sha3-256": "784335e2ae23886cb5fa1261fc3dfbaee12623241791c5e4d78b0da619a78051",
		"sha3-512": "bce76c1eacfaf74912144f26e0fdadba5f7b6893fb046e21d280ffeb3f1f1bf14213862e292e3be64be8c6e5c8216b839c658f3893eae700e4a92f5625ec25c9",
		"sha3-224": "6ef5918377a5309c4b8b41a4a1d9c680cc3259e7a7619f47ca345714",
		"sha3-384": "ba6ea7b48af10d7025c4b0c6a105f410278705020d921377c729fe41e88cd9fc2b851002b4cc5a42ba5c34ca8a07b36d",
		"blake2s":  "7cd93f6d174040f3618982922701c54ec5b02dd28902b5160628b1d5516a62c9",
		"blake2b":  "ca0dbbe27fca7e5d97b612a76b66d9d42fd67ece4265a50c09ccaefcdc03d9d5a87fa1fddc926ae10c6667342c69df5c33117cf636fca82ac1377c2b4e23e2bc",
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
		Extra: map[string]any{
			"Foo": "foo",
			"docker": config.Docker{
				ID:  "id",
				Use: "docker",
			},
			"fail-plz": config.Homebrew{
				Service: "aaaa",
			},
			"unsupported":      func() {},
			"binaries":         []string{"foo", "bar"},
			"docker_unmarshal": map[string]any{"id": "foo"}, // this is what it would look like loading from json
		},
	}

	t.Run("string", func(t *testing.T) {
		foo := MustExtra[string](a, "Foo")
		require.Equal(t, "foo", foo)
		require.Equal(t, "foo", ExtraOr(a, "Foo", "bar"))
	})

	t.Run("missing field", func(t *testing.T) {
		require.Equal(t, "bar", ExtraOr(a, "Foobar", "bar"))
		require.PanicsWithError(t, "extra: Foobar: key not present", func() {
			MustExtra[string](a, "Foobar")
		})
	})

	t.Run("complex", func(t *testing.T) {
		docker := MustExtra[config.Docker](a, "docker")
		require.Equal(t, "id", docker.ID)
	})

	t.Run("array", func(t *testing.T) {
		require.Equal(t, []string{"foo", "bar"}, ExtraOr(a, "binaries", []string{}))
		require.Equal(t, []string{"foo", "bar"}, MustExtra[[]string](a, "binaries"))
	})

	t.Run("unmarshal complex", func(t *testing.T) {
		expected := config.Docker{ID: "foo"}
		require.Equal(t, expected, ExtraOr(a, "docker_unmarshal", config.Docker{}))
		require.Equal(t, expected, MustExtra[config.Docker](a, "docker_unmarshal"))
	})

	t.Run("unmarshal error", func(t *testing.T) {
		errString := "extra: fail-plz: json: unknown field \"repository\""
		require.PanicsWithError(t, errString, func() {
			MustExtra[config.Docker](a, "fail-plz")
		})
		require.PanicsWithError(t, errString, func() {
			ExtraOr(a, "fail-plz", config.Docker{})
		})
	})

	t.Run("marshal error", func(t *testing.T) {
		errString := "extra: unsupported: json: unsupported type: func()"
		require.PanicsWithError(t, errString, func() {
			MustExtra[string](a, "unsupported")
		})
		require.PanicsWithError(t, errString, func() {
			ExtraOr(a, "unsupported", "")
		})
	})
}

func TestByIDs(t *testing.T) {
	data := []*Artifact{
		{
			Name: "foo",
			Extra: map[string]any{
				ExtraID: "foo",
			},
		},
		{
			Name: "bar",
			Extra: map[string]any{
				ExtraID: "bar",
			},
		},
		{
			Name: "foobar",
			Extra: map[string]any{
				ExtraID: "foo",
			},
		},
		{
			Name: "check",
			Extra: map[string]any{
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
			Extra: map[string]any{
				ExtraExt: ".deb",
			},
		},
		{
			Name: "bar",
			Extra: map[string]any{
				ExtraExt: "deb",
			},
		},
		{
			Name: "foobar",
			Extra: map[string]any{
				ExtraExt: "rpm",
			},
		},
		{
			Name:  "check",
			Extra: map[string]any{},
		},
	}
	artifacts := New()
	for _, a := range data {
		artifacts.Add(a)
	}

	require.Len(t, artifacts.Filter(ByExt("deb")).items, 2)
	require.Len(t, artifacts.Filter(ByExt("rpm")).items, 1)
	require.Len(t, artifacts.Filter(ByExt("rpm", ".deb")).items, 3)
	require.Empty(t, artifacts.Filter(ByExt("foo")).items)
}

func TestByFormats(t *testing.T) {
	data := []*Artifact{
		{
			Name: "foo",
			Extra: map[string]any{
				ExtraFormat: "zip",
			},
		},
		{
			Name: "bar",
			Extra: map[string]any{
				ExtraFormat: "tar.gz",
			},
		},
		{
			Name: "foobar",
			Extra: map[string]any{
				ExtraFormat: "zip",
			},
		},
		{
			Name: "bin",
			Extra: map[string]any{
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
			Extra: map[string]any{
				"Refresh": func() error {
					return os.WriteFile(path, []byte("hello"), 0o765)
				},
			},
		})
		artifacts.Add(&Artifact{
			Name: "no refresh",
			Type: Checksum,
		})

		require.NoError(t, artifacts.Refresh())

		bts, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "hello", string(bts))
	})

	t.Run("nok", func(t *testing.T) {
		artifacts := New()
		artifacts.Add(&Artifact{
			Name: "fail",
			Type: Checksum,
			Extra: map[string]any{
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
			Extra: map[string]any{
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
		require.EqualError(t, artifacts.Visit(func(_ *Artifact) error {
			return fmt.Errorf("fake err")
		}), `fake err`)
	})
}

func TestMarshalJSON(t *testing.T) {
	artifacts := New()
	artifacts.Add(&Artifact{
		Name: "foo",
		Type: Binary,
		Extra: map[string]any{
			ExtraID: "adsad",
		},
	})
	artifacts.Add(&Artifact{
		Name: "foo",
		Type: UploadableArchive,
		Extra: map[string]any{
			ExtraID: "adsad",
		},
	})
	artifacts.Add(&Artifact{
		Name: "foo",
		Type: Checksum,
		Extra: map[string]any{
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

func TestArtifactTypeStringer(t *testing.T) {
	for i := 1; i < int(lastMarker); i++ {
		t.Run(fmt.Sprintf("type-%d-%s", i, Type(i).String()), func(t *testing.T) {
			require.NotEqual(t, "unknown", Type(i).String())
		})
	}

	t.Run("unknown", func(t *testing.T) {
		require.Equal(t, "unknown", Type(99999).String())
	})
}

func TestArtifactTypeIsUploadable(t *testing.T) {
	nonUploadable := []Type{
		Binary,
		Metadata,
		SrcInfo,
		SourceSrcInfo,
		PkgBuild,
		SourcePkgBuild,
		UniversalBinary,
	}
	for i := range lastMarker - 1 {
		up := i.isUploadable()
		t.Run(fmt.Sprintf("%s-%v", i.String(), up), func(t *testing.T) {
			if slices.Contains(nonUploadable, i) {
				require.False(t, up)
				return
			}
			require.True(t, up)
		})
	}
}
