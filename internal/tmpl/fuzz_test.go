package tmpl

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func FuzzTemplateApplier(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{ProjectName: "test"})
		tpl := New(ctx)
		_, err := tpl.Apply(data)
		if err == nil {
			return
		}
		require.ErrorAs(t, err, &Error{})
	})
}

func FuzzTemplateWithArtifact(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{ProjectName: "test"})
		tpl := New(ctx).WithArtifact(&artifact.Artifact{
			Name:   "test",
			Path:   "fake-filename.bin",
			Goarch: "amd64",
			Goos:   "linux",
			Target: "linux_amd64",
		})

		_, err := tpl.Apply(data)
		if err == nil {
			return
		}
		require.ErrorAs(t, err, &Error{})
	})
}

func FuzzTemplateBool(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		ctx := testctx.Wrap(t.Context())
		tpl := New(ctx)
		_, err := tpl.Apply(data)
		if err == nil {
			return
		}
		require.ErrorAs(t, err, &Error{})
	})
}

func FuzzTemplateSlice(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		ctx := testctx.Wrap(t.Context())
		tpl := New(ctx)
		_, err := tpl.Slice([]string{data})
		if err == nil {
			return
		}
		require.ErrorAs(t, err, &Error{})
	})
}

func FuzzTemplateWithBuildOptions(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		ctx := testctx.Wrap(t.Context())
		target := &buildTarget{
			Target: "linux_amd64",
			Goos:   "linux",
			Goarch: "amd64",
		}

		tpl := New(ctx).WithBuildOptions(build.Options{
			Name:   "test",
			Target: target,
		})

		_, err := tpl.Apply(data)
		if err == nil {
			return
		}
		require.ErrorAs(t, err, &Error{})
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
