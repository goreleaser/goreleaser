package bottle

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/golden"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestTagsFor(t *testing.T) {
	for in, want := range map[string][]string{
		"darwin/arm64":  {"arm64_monterey"},
		"darwin/amd64":  {"monterey"},
		"darwin/all":    {"arm64_monterey", "monterey"},
		"linux/arm64":   {"arm64_linux"},
		"linux/amd64":   {"x86_64_linux"},
		"linux/arm":     nil,
		"windows/amd64": nil,
	} {
		goos, goarch, _ := cut(in)
		require.Equal(t, want, tagsFor(goos, goarch, "monterey"), in)
	}
}

func cut(s string) (string, string, bool) {
	for i := range s {
		if s[i] == '/' {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}

func TestFormulaClass(t *testing.T) {
	// Same expectations as Homebrew's Formulary.class_s.
	for in, want := range map[string]string{
		"foo":               "Foo",
		"foo-bar":           "FooBar",
		"foo_bar.baz":       "FooBarBaz",
		"foo@1":             "FooAT1",
		"foo@bar":           "Foo@bar",
		"c++":               "Cxx",
		"Some-Thing":        "SomeThing",
		"foo_bar@v6.0.0-rc": "FooBar@v600Rc",
	} {
		require.Equal(t, want, formulaClass(in), in)
	}
}

func TestRubyQuote(t *testing.T) {
	require.Equal(t, `"a \"b\" \\ \#{x}\n"`, rubyQuote("a \"b\" \\ #{x}\n"))
}

// setup builds a context with binaries for every bottle platform plus a
// windows one that must be ignored, and a source archive.
func setup(t *testing.T, cfg config.HomebrewBottle, withSource bool) *context.Context {
	t.Helper()
	dist := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "foo",
		Dist:        dist,
		Builds: []config.Build{{
			ID:   "foo",
			Main: "./cmd/foo",
			BuildDetails: config.BuildDetails{
				Ldflags: []string{"-s -w -X main.version={{ .Version }}"},
				Env:     []string{"CGO_ENABLED=0"},
			},
		}},
		HomebrewBottles: []config.HomebrewBottle{cfg},
	}, testctx.WithVersion("1.2.3"), testctx.WithCurrentTag("v1.2.3"))

	for _, p := range []struct{ goos, goarch string }{
		{"darwin", "arm64"},
		{"darwin", "amd64"},
		{"linux", "arm64"},
		{"linux", "amd64"},
		{"windows", "amd64"},
	} {
		path := filepath.Join(dist, p.goos+"_"+p.goarch, "foo")
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(p.goos+p.goarch), 0o755))
		art := &artifact.Artifact{
			Type:   artifact.Binary,
			Name:   "foo",
			Path:   path,
			Goos:   p.goos,
			Goarch: p.goarch,
			Extra: map[string]any{
				artifact.ExtraID:     "foo",
				artifact.ExtraBinary: "foo",
			},
		}
		if p.goarch == "amd64" {
			art.Goamd64 = "v1"
		}
		ctx.Artifacts.Add(art)
	}
	if withSource {
		path := filepath.Join(dist, "foo-1.2.3.tar.gz")
		require.NoError(t, os.WriteFile(path, []byte("source"), 0o644))
		ctx.Artifacts.Add(&artifact.Artifact{
			Type: artifact.UploadableSourceArchive,
			Name: "foo-1.2.3.tar.gz",
			Path: path,
		})
	}
	require.NoError(t, Archive{}.Default(ctx))
	return ctx
}

func baseConfig() config.HomebrewBottle {
	return config.HomebrewBottle{
		Repository:  config.RepoRef{Owner: "blairham", Name: "homebrew-tap"},
		Description: `Run "foo" fast`,
		Homepage:    "https://example.com",
		License:     "MIT",
		Test:        `system "#{bin}/foo", "--version"`,
	}
}

func TestArchiveAndFormula(t *testing.T) {
	ctx := setup(t, baseConfig(), true)
	require.NoError(t, Archive{}.Run(ctx))

	bottles := ctx.Artifacts.Filter(artifact.ByType(artifact.HomebrewBottle)).List()
	names := make([]string, 0, len(bottles))
	for _, b := range bottles {
		names = append(names, b.Name)
	}
	require.ElementsMatch(t, []string{
		"foo-1.2.3.arm64_monterey.bottle.tar.gz",
		"foo-1.2.3.monterey.bottle.tar.gz",
		"foo-1.2.3.arm64_linux.bottle.tar.gz",
		"foo-1.2.3.x86_64_linux.bottle.tar.gz",
	}, names)

	for _, b := range bottles {
		if b.Name != "foo-1.2.3.arm64_linux.bottle.tar.gz" {
			continue
		}
		require.Equal(t, map[string]string{
			"foo/":              "",
			"foo/1.2.3/":        "",
			"foo/1.2.3/bin/":    "",
			"foo/1.2.3/bin/foo": "linuxarm64",
		}, untar(t, b.Path))
	}

	require.NoError(t, doRun(ctx, ctx.Config.HomebrewBottles[0], client.NewMock()))
	formulas := ctx.Artifacts.Filter(artifact.ByType(artifact.HomebrewBottleFormula)).List()
	require.Len(t, formulas, 1)
	content, err := os.ReadFile(formulas[0].Path)
	require.NoError(t, err)
	// checksums are of fixture bytes, so they are stable.
	golden.RequireEqualRb(t, content)
}

func TestFormulaCustomInstall(t *testing.T) {
	cfg := baseConfig()
	cfg.Install = `system "make", "install", "PREFIX=#{prefix}"`
	cfg.ExtraInstall = `generate_completions_from_executable(bin/"foo", "completion")`
	ctx := setup(t, cfg, true)
	require.NoError(t, Archive{}.Run(ctx))
	out, err := buildFormula(ctx, ctx.Config.HomebrewBottles[0], client.NewMock())
	require.NoError(t, err)
	require.Contains(t, out, `  def install
    system "make", "install", "PREFIX=#{prefix}"
    generate_completions_from_executable(bin/"foo", "completion")
  end`)
	require.NotContains(t, out, `depends_on "go"`)
}

func TestFormulaRequiresSource(t *testing.T) {
	ctx := setup(t, baseConfig(), false)
	require.NoError(t, Archive{}.Run(ctx))
	_, err := buildFormula(ctx, ctx.Config.HomebrewBottles[0], client.NewMock())
	require.ErrorIs(t, err, ErrNoSourceArchive)
}

func TestBottleExtraFiles(t *testing.T) {
	cfg := baseConfig()
	src := filepath.Join(t.TempDir(), "_foo")
	require.NoError(t, os.WriteFile(src, []byte("#compdef foo"), 0o644))
	cfg.IDs = []string{"foo"}
	cfg.Bottles.Files = []config.HomebrewBottleFile{{Src: src, Dst: "share/zsh/site-functions/_foo"}}
	ctx := setup(t, cfg, true)
	require.NoError(t, Archive{}.Run(ctx))
	b := ctx.Artifacts.Filter(artifact.ByType(artifact.HomebrewBottle)).List()[0]
	files := untar(t, b.Path)
	require.Equal(t, "#compdef foo", files["foo/1.2.3/share/zsh/site-functions/_foo"])
	require.Contains(t, files, "foo/1.2.3/share/zsh/")
}

func TestNoBinaries(t *testing.T) {
	cfg := baseConfig()
	cfg.IDs = []string{"nope"}
	ctx := setup(t, cfg, true)
	require.ErrorIs(t, Archive{}.Run(ctx), ErrNoBinaries)
}

func TestSkip(t *testing.T) {
	require.True(t, Archive{}.Skip(testctx.Wrap(t.Context())))
	require.True(t, Formula{}.Skip(testctx.Wrap(t.Context())))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{HomebrewBottles: []config.HomebrewBottle{{}}})
	require.False(t, Archive{}.Skip(ctx))
	require.False(t, Formula{}.Skip(ctx))
}

func untar(t *testing.T, path string) map[string]string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	gr, err := gzip.NewReader(f)
	require.NoError(t, err)
	tr := tar.NewReader(gr)
	out := map[string]string{}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return out
		}
		require.NoError(t, err)
		b, err := io.ReadAll(tr)
		require.NoError(t, err)
		if h.Typeflag == tar.TypeReg {
			want := int64(0o644)
			if filepath.Base(filepath.Dir(h.Name)) == "bin" {
				want = 0o755
			}
			require.Equal(t, want, h.Mode, h.Name)
		}
		out[h.Name] = string(b)
	}
}
