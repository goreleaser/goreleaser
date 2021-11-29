package release

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescribeBody(t *testing.T) {
	changelog := "feature1: description\nfeature2: other description"
	ctx := context.New(config.Project{})
	ctx.ReleaseNotes = changelog
	out, err := describeBody(ctx)
	require.NoError(t, err)

	golden.RequireEqual(t, out.Bytes())
}

func TestDontEscapeHTML(t *testing.T) {
	changelog := "<h1>test</h1>"
	ctx := context.New(config.Project{})
	ctx.ReleaseNotes = changelog

	out, err := describeBody(ctx)
	require.NoError(t, err)
	require.Contains(t, out.String(), changelog)
}

func TestDescribeBodyWithHeaderAndFooter(t *testing.T) {
	changelog := "feature1: description\nfeature2: other description"
	ctx := context.New(config.Project{
		Release: config.Release{
			Header: "## Yada yada yada\nsomething\n",
			Footer: "\n---\n\nGet images at docker.io/foo/bar:{{.Tag}}\n\n---\n\nGet GoReleaser Pro at https://goreleaser.com/pro",
		},
	})
	ctx.ReleaseNotes = changelog
	ctx.Git = context.GitInfo{CurrentTag: "v1.0"}
	out, err := describeBody(ctx)
	require.NoError(t, err)

	golden.RequireEqual(t, out.Bytes())
}

func TestDescribeBodyWithInvalidHeaderTemplate(t *testing.T) {
	ctx := context.New(config.Project{
		Release: config.Release{
			Header: "## {{ .Nop }\n",
		},
	})
	_, err := describeBody(ctx)
	require.EqualError(t, err, `template: tmpl:1: unexpected "}" in operand`)
}

func TestDescribeBodyWithInvalidFooterTemplate(t *testing.T) {
	ctx := context.New(config.Project{
		Release: config.Release{
			Footer: "{{ .Nops }",
		},
	})
	_, err := describeBody(ctx)
	require.EqualError(t, err, `template: tmpl:1: unexpected "}" in operand`)
}
