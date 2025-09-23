package tmpl

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
)

func FuzzTemplateApplier(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		ctx := testctx.NewWithCfg(config.Project{ProjectName: "test"})
		tpl := New(ctx)
		_, _ = tpl.Apply(data)
	})
}

func FuzzTemplateWithArtifact(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		ctx := testctx.NewWithCfg(config.Project{ProjectName: "test"})
		tpl := New(ctx).WithArtifact(&artifact.Artifact{
			Name:   "test",
			Path:   "fake-filename.bin",
			Goarch: "amd64",
			Goos:   "linux",
			Target: "linux_amd64",
		})

		_, _ = tpl.Apply(data)
	})
}

func FuzzTemplateBool(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		ctx := testctx.New()
		tpl := New(ctx)
		_, _ = tpl.Bool(data)
	})
}

func FuzzTemplateSlice(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		ctx := testctx.New()
		tpl := New(ctx)
		input := []string{data}
		_, _ = tpl.Slice(input)
	})
}

func FuzzTemplateWithBuildOptions(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		ctx := testctx.New()
		target := &buildTarget{
			Target: "linux_amd64",
			Goos:   "linux",
			Goarch: "amd64",
		}

		tpl := New(ctx).WithBuildOptions(build.Options{
			Name:   "test",
			Target: target,
		})

		_, _ = tpl.Apply(data)
	})
}

type buildTarget struct {
	Target string
	Goos   string
	Goarch string
}

func (t *buildTarget) String() string { return t.Target }

func (t *buildTarget) Fields() map[string]string {
	return map[string]string{
		"target": t.Target,
		"os":     t.Goos,
		"arch":   t.Goarch,
	}
}
