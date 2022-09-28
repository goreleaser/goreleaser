package srpm

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestRunPipe(t *testing.T) {
	// Setup a context with a source archive.
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	sourceArchivePath := filepath.Join(dist, "example-1.0.0.tar.gz")
	f, err := os.Create(sourceArchivePath)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "example",
		Dist:        dist,
		SRPM: config.SRPM{
			NFPMRPM: config.NFPMRPM{
				Summary: "Example summary",
			},
			Enabled:         true,
			ImportPath:      "github.com/example/example",
			License:         "MIT",
			LicenseFileName: "LICENSE",
			Packager:        "Example packager",
			Vendor:          "Example vendor",
			URL:             "https://example.com",
			Description:     "Example description",
			Docs: []string{
				"README.md",
			},
			SpecTemplateFile: "testdata/example.spec.tmpl",
		},
	})
	ctx.Version = "1.0.0"
	ctx.Git = context.GitInfo{
		FullCommit: "e070258c90772fbcf1cb94c2b937ff25a011b5c8",
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "example-1.0.0.tar.gz",
		Path: sourceArchivePath,
		Type: artifact.UploadableSourceArchive,
	})

	// Run the source RPM pipe.
	var pipe Pipe
	require.NoError(t, pipe.Default(ctx))
	require.NoError(t, pipe.Run(ctx))

	// Check the source RPM artifact.
	sourceRPMs := ctx.Artifacts.Filter(artifact.ByType(artifact.SourceRPM)).List()
	require.Len(t, sourceRPMs, 1)
	sourceRPM := sourceRPMs[0]
	require.Equal(t, "example-1.0.0.src.rpm", sourceRPM.Name)
	require.Equal(t, filepath.ToSlash(filepath.Join(dist, "example-1.0.0.src.rpm")), sourceRPM.Path)
	// FIXME check source RPM contents using https://github.com/sassoftware/go-rpmutils?
	// FIXME check source RPM contents using https://github.com/cavaliergopher/rpm?

	// Check the .spec artifact.
	rpmSpecs := ctx.Artifacts.Filter(artifact.ByType(artifact.RPMSpec)).List()
	require.Len(t, rpmSpecs, 1)
	rpmSpecContents, err := os.ReadFile(rpmSpecs[0].Path)
	require.NoError(t, err)
	require.True(t, regexp.MustCompile(`(?m)^%global\s+goipath\s+github.com/example/example$`).Match(rpmSpecContents))
	require.True(t, regexp.MustCompile(`(?m)^%global\s+commit\s+e070258c90772fbcf1cb94c2b937ff25a011b5c8$`).Match(rpmSpecContents))
	require.True(t, regexp.MustCompile(`(?m)^%global\s+golicenses\s+LICENSE$`).Match(rpmSpecContents))
	require.True(t, regexp.MustCompile(`(?m)^%global\s+godocs\s+README\.md$`).Match(rpmSpecContents))
	require.True(t, regexp.MustCompile(`(?m)^Version:\s+1\.0\.0$`).Match(rpmSpecContents))
	require.True(t, regexp.MustCompile(`(?m)^Summary:\s+Example summary$`).Match(rpmSpecContents))
	require.True(t, regexp.MustCompile(`(?m)^%doc\s+README\.md$`).Match(rpmSpecContents))
	require.True(t, regexp.MustCompile(`(?m)^%license\s+LICENSE$`).Match(rpmSpecContents))
	// FIXME add tests for all remaining configurable fields
}
