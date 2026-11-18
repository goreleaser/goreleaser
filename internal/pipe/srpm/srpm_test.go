package srpm

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/snapshot"
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
			Summary:         "Example summary",
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

func TestRunPipeSnapshotVersion(t *testing.T) {
	for _, tt := range []struct {
		name    string
		version string
		wantErr bool
	}{
		{name: "default snapshot", version: "1.0.0-SNAPSHOT-abc123"},
		{name: "non semver snapshot", version: "nightly"},
		{name: "build metadata", version: "1.0.0-dev+abc123"},
		{name: "shell macro", version: "0.2.0%(echo safe-marker)", wantErr: true},
		{name: "lua macro", version: "0.2.0%{lua:print(1)}", wantErr: true},
		{name: "escaped macro", version: "0.2.0%%(echo safe-marker)", wantErr: true},
		{name: "bare percent", version: "0.2.0%", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			folder := t.TempDir()
			dist := filepath.Join(folder, "dist")
			sourcePath := filepath.Join(folder, "source.tar.gz")
			require.NoError(t, os.WriteFile(sourcePath, nil, 0o644))
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				ProjectName: "example",
				Dist:        dist,
				Snapshot: config.Snapshot{
					VersionTemplate: "{{ .Env.SNAPSHOT_VERSION }}",
				},
				SRPM: config.SRPM{
					Enabled:    true,
					Summary:    "Example summary",
					License:    "MIT",
					ImportPath: "github.com/example/example",
					SpecFile:   "testdata/example.spec.tmpl",
				},
			}, testctx.Snapshot)
			ctx.Env["SNAPSHOT_VERSION"] = tt.version
			require.NoError(t, snapshot.Pipe{}.Run(ctx))
			require.Equal(t, tt.version, ctx.Version)
			sourceName := "example-" + ctx.Version + ".tar.gz"
			ctx.Artifacts.Add(&artifact.Artifact{
				Name: sourceName,
				Path: sourcePath,
				Type: artifact.UploadableSourceArchive,
			})

			require.NoError(t, Pipe{}.Default(ctx))
			err := Pipe{}.Run(ctx)
			specPath := filepath.Join(dist, "example.srpm.spec")
			if tt.wantErr {
				if err == nil {
					spec, readErr := os.ReadFile(specPath)
					require.NoError(t, readErr)
					for line := range strings.SplitSeq(string(spec), "\n") {
						if strings.HasPrefix(line, "Version:") || strings.HasPrefix(line, "Source:") {
							t.Log(line)
						}
					}
				}
				require.EqualError(t, err, "source RPM version must not contain '%'")
				require.NoDirExists(t, dist)
				require.Empty(t, ctx.Artifacts.Filter(artifact.ByType(artifact.SourceRPM)).List())
				return
			}

			require.NoError(t, err)
			spec, err := os.ReadFile(specPath)
			require.NoError(t, err)
			require.Contains(t, string(spec), "Version:        "+tt.version+"\n")
			require.Contains(t, string(spec), "Source:         "+sourceName+"\n")
			sourceRPMs := ctx.Artifacts.Filter(artifact.ByType(artifact.SourceRPM)).List()
			require.Len(t, sourceRPMs, 1)
			require.Equal(t, "example-"+tt.version+".src.rpm", sourceRPMs[0].Name)
		})
	}
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
			Summary:          "Example summary",
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
			Summary:    "Example summary",
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
			Summary:    "Example summary",
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
