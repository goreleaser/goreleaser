package release

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescribeBody(t *testing.T) {
	changelog := "feature1: description\nfeature2: other description"
	ctx := testctx.New()
	ctx.ReleaseNotes = changelog
	out, err := describeBody(ctx)
	require.NoError(t, err)

	golden.RequireEqual(t, out.Bytes())
}

func TestDontEscapeHTML(t *testing.T) {
	changelog := "<h1>test</h1>"
	ctx := testctx.New()
	ctx.ReleaseNotes = changelog

	out, err := describeBody(ctx)
	require.NoError(t, err)
	require.Contains(t, out.String(), changelog)
}

func TestDescribeBodyWithHeaderAndFooter(t *testing.T) {
	changelog := "feature1: description\nfeature2: other description"
	ctx := testctx.NewWithCfg(
		config.Project{
			Release: config.Release{
				Header: "## Yada yada yada\nsomething\n",
				Footer: `
---

Get images at docker.io/foo/bar:{{.Tag}}

---

Get GoReleaser Pro at https://goreleaser.com/pro

---

## Checksums

` + "```\n{{ .Checksums }}\n```" + `
				`,
			},
		},
		testctx.WithCurrentTag("v1.0"),
		func(ctx *context.Context) { ctx.ReleaseNotes = changelog },
	)

	checksumPath := filepath.Join(t.TempDir(), "checksums.txt")
	checksumContent := "f674623cf1edd0f753e620688cedee4e7c0e837ac1e53c0cbbce132ffe35fd52  foo.zip"
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "checksums.txt",
		Path: checksumPath,
		Type: artifact.Checksum,
		Extra: map[string]interface{}{
			artifact.ExtraRefresh: func() error {
				return os.WriteFile(checksumPath, []byte(checksumContent), 0o644)
			},
		},
	})
	out, err := describeBody(ctx)
	require.NoError(t, err)

	golden.RequireEqual(t, out.Bytes())
}

func TestDescribeBodyWithInvalidHeaderTemplate(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Release: config.Release{
			Header: "## {{ .Nop }\n",
		},
	})
	_, err := describeBody(ctx)
	testlib.RequireTemplateError(t, err)
}

func TestDescribeBodyWithInvalidFooterTemplate(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Release: config.Release{
			Footer: "{{ .Nops }",
		},
	})
	_, err := describeBody(ctx)
	testlib.RequireTemplateError(t, err)
}
