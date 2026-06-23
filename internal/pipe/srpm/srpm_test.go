package srpm

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{})
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("skip flag", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			SRPM: config.SRPM{Enabled: true},
		}, testctx.Skip(skips.SRPM))
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("enabled", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			SRPM: config.SRPM{Enabled: true},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "example",
		SRPM: config.SRPM{
			Enabled: true,
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "example", ctx.Config.SRPM.PackageName)
	require.Equal(t, defaultFileNameTemplate, ctx.Config.SRPM.FileNameTemplate)
	require.Equal(t, map[string]string{"example": "%{goipath}"}, ctx.Config.SRPM.Bins)
}

func TestRunPipe(t *testing.T) {
	// Setup a context with a source archive.
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	sourceArchivePath := filepath.Join(dist, "example-1.0.0.tar.gz")
	f, err := os.Create(sourceArchivePath)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
			SpecFile: "testdata/example.spec.tmpl",
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

	var pipe Pipe
	require.NoError(t, pipe.Default(ctx))
	require.NoError(t, pipe.Run(ctx))

	// Check the source RPM artifact.
	sourceRPMs := ctx.Artifacts.Filter(artifact.ByType(artifact.SourceRPM)).List()
	require.Len(t, sourceRPMs, 1)
	sourceRPM := sourceRPMs[0]
	require.Equal(t, "example-1.0.0.src.rpm", sourceRPM.Name)
	require.Equal(t, filepath.ToSlash(filepath.Join(dist, "example-1.0.0.src.rpm")), sourceRPM.Path)
	require.Equal(t, "src.rpm", sourceRPM.Format())
	require.Equal(t, ".src.rpm", sourceRPM.Ext())

	// Check the .spec artifact.
	rpmSpecContents, err := os.ReadFile(filepath.Join(dist, "example.srpm.spec"))
	require.NoError(t, err)
	require.True(t, regexp.MustCompile(`(?m)^%global\s+goipath\s+github\.com/example/example$`).Match(rpmSpecContents))
	require.True(t, regexp.MustCompile(`(?m)^%global\s+commit\s+e070258c90772fbcf1cb94c2b937ff25a011b5c8$`).Match(rpmSpecContents))
	require.True(t, regexp.MustCompile(`(?m)^%global\s+golicenses\s+LICENSE$`).Match(rpmSpecContents))
	require.True(t, regexp.MustCompile(`(?m)^%global\s+godocs\s+.*README\.md`).Match(rpmSpecContents))
	require.True(t, regexp.MustCompile(`(?m)^Version:\s+1\.0\.0$`).Match(rpmSpecContents))
	require.True(t, regexp.MustCompile(`(?m)^Summary:\s+Example summary$`).Match(rpmSpecContents))
	require.True(t, regexp.MustCompile(`(?m)^%doc\s+README\.md$`).Match(rpmSpecContents))
	require.True(t, regexp.MustCompile(`(?m)^%license\s+LICENSE$`).Match(rpmSpecContents))
}

func TestRunPipeConventionalFileName(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	sourceArchivePath := filepath.Join(dist, "example-1.0.0.tar.gz")
	f, err := os.Create(sourceArchivePath)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "example",
		Dist:        dist,
		SRPM: config.SRPM{
			NFPMRPM:          config.NFPMRPM{Summary: "Example summary"},
			Enabled:          true,
			ImportPath:       "github.com/example/example",
			License:          "MIT",
			SpecFile:         "testdata/example.spec.tmpl",
			FileNameTemplate: "{{.ConventionalFileName}}",
		},
	})
	ctx.Version = "1.0.0"
	ctx.Git = context.GitInfo{FullCommit: "abc123"}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "example-1.0.0.tar.gz",
		Path: sourceArchivePath,
		Type: artifact.UploadableSourceArchive,
	})

	var pipe Pipe
	require.NoError(t, pipe.Default(ctx))
	require.NoError(t, pipe.Run(ctx))

	sourceRPMs := ctx.Artifacts.Filter(artifact.ByType(artifact.SourceRPM)).List()
	require.Len(t, sourceRPMs, 1)
	require.True(t, strings.HasSuffix(sourceRPMs[0].Name, ".src.rpm"), "expected .src.rpm suffix, got %q", sourceRPMs[0].Name)
}

func TestRunPipeNoSourceArchive(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "example",
		Dist:        t.TempDir(),
		SRPM: config.SRPM{
			Enabled:  true,
			SpecFile: "testdata/example.spec.tmpl",
		},
	})
	ctx.Version = "1.0.0"
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Run(ctx), "no source archives found")
}

func TestRunPipeContentsTemplates(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	sourceArchivePath := filepath.Join(dist, "example-1.0.0.tar.gz")
	f, err := os.Create(sourceArchivePath)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	// Create an extra file to include via contents.
	extraPatch := filepath.Join(folder, "fix-build.patch")
	require.NoError(t, os.WriteFile(extraPatch, []byte("--- a\n+++ b\n"), 0o644))

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "example",
		Dist:        dist,
		SRPM: config.SRPM{
			NFPMRPM:    config.NFPMRPM{Summary: "Example summary"},
			Enabled:    true,
			ImportPath: "github.com/example/example",
			License:    "MIT",
			SpecFile:   "testdata/example.spec.tmpl",
			Contents: []config.NFPMContent{
				{
					Source:      "{{ .Env.PATCH_FILE }}",
					Destination: "fix-build-{{ .Version }}.patch",
					FileInfo: config.FileInfo{
						Owner: "{{ .Env.PATCH_OWNER }}",
						Group: "mock",
					},
				},
			},
		},
	})
	ctx.Version = "1.0.0"
	ctx.Git = context.GitInfo{FullCommit: "abc123"}
	ctx.Env["PATCH_FILE"] = extraPatch
	ctx.Env["PATCH_OWNER"] = "mockbuild"
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "example-1.0.0.tar.gz",
		Path: sourceArchivePath,
		Type: artifact.UploadableSourceArchive,
	})

	var pipe Pipe
	require.NoError(t, pipe.Default(ctx))
	require.NoError(t, pipe.Run(ctx))

	sourceRPMs := ctx.Artifacts.Filter(artifact.ByType(artifact.SourceRPM)).List()
	require.Len(t, sourceRPMs, 1)
}

func TestRunPipeContentsInvalidMTime(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	sourceArchivePath := filepath.Join(dist, "example-1.0.0.tar.gz")
	f, err := os.Create(sourceArchivePath)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "example",
		Dist:        dist,
		SRPM: config.SRPM{
			NFPMRPM:    config.NFPMRPM{Summary: "Example summary"},
			Enabled:    true,
			ImportPath: "github.com/example/example",
			License:    "MIT",
			SpecFile:   "testdata/example.spec.tmpl",
			Contents: []config.NFPMContent{
				{
					Source:      "README.md",
					Destination: "README.md",
					FileInfo: config.FileInfo{
						MTime: "not-a-valid-time",
					},
				},
			},
		},
	})
	ctx.Version = "1.0.0"
	ctx.Git = context.GitInfo{FullCommit: "abc123"}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "example-1.0.0.tar.gz",
		Path: sourceArchivePath,
		Type: artifact.UploadableSourceArchive,
	})

	var pipe Pipe
	require.NoError(t, pipe.Default(ctx))
	require.ErrorContains(t, pipe.Run(ctx), "failed to parse not-a-valid-time")
}
