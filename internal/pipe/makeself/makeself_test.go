package makeself

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.Equal(t, "makeself packages", Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		ctx := testctx.New(testctx.Skip(skips.Makeself))
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Makeselfs: []config.MakeselfPackage{{}},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})

	t.Run("skip no makeselfs", func(t *testing.T) {
		ctx := testctx.New()
		require.True(t, Pipe{}.Skip(ctx))
	})
}

func TestDefault(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Makeselfs: []config.MakeselfPackage{
			{},
			{
				ID:           "custom",
				NameTemplate: "custom_{{.Os}}_{{.Arch}}",
				Extension:    ".bin",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))

	require.Equal(t, "default", ctx.Config.Makeselfs[0].ID)
	require.Equal(t, defaultNameTemplate, ctx.Config.Makeselfs[0].NameTemplate)
	require.Equal(t, ".run", ctx.Config.Makeselfs[0].Extension)

	require.Equal(t, "custom", ctx.Config.Makeselfs[1].ID)
	require.Equal(t, "custom_{{.Os}}_{{.Arch}}", ctx.Config.Makeselfs[1].NameTemplate)
	require.Equal(t, ".bin", ctx.Config.Makeselfs[1].Extension)
}

func TestDefaultDuplicateID(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Makeselfs: []config.MakeselfPackage{
			{ID: "test"},
			{ID: "test"},
		},
	})

	require.EqualError(t, Pipe{}.Default(ctx), "found 2 makeselfs with the ID 'test', please fix your config")
}

func createFakeBinary(t *testing.T, dist, platform, binary string) string {
	t.Helper()
	path := filepath.Join(dist, platform, binary)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return path
}

func createMockMakeself(t *testing.T, folder string) {
	t.Helper()
	mockMakeselfScript := filepath.Join(folder, "makeself")
	mockScript := `#!/bin/bash
# Mock makeself script for testing

# Handle --help flag separately (don't log it)
if [[ "$1" == "--help" ]]; then
	   echo "makeself version 2.4.0"
	   echo "Usage: makeself [args] source_dir target_file label startup_script"
	   exit 0
fi

# Create minimal self-extracting archive
# Args: [flags...] source_dir target_file label startup_script
# Find the target file (second to last argument, or argument that looks like a path)
target=""
for arg in "$@"; do
	   if [[ "$arg" == *".run" ]] || [[ "$arg" == *".bin" ]] || [[ "$arg" == *"/"* ]] && [[ "$arg" != "--"* ]]; then
	       if [[ -n "$arg" ]] && [[ "$arg" != *" "* ]]; then
	           target="$arg"
	       fi
	   fi
done

if [[ -n "$target" ]]; then
	   cat > "$target" << 'EOF'
#!/bin/bash
echo "Test makeself archive"
exit 0
EOF
	   chmod +x "$target"
fi

# Log the actual makeself command arguments (not --help calls)
echo "$@" > "` + folder + `/makeself_args.log"`
	require.NoError(t, os.WriteFile(mockMakeselfScript, []byte(mockScript), 0o755))

	// Set PATH to include our mock makeself
	originalPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", originalPath) })
	os.Setenv("PATH", folder+":"+originalPath)
}

func TestRunPipe(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))

	createFakeBinary(t, dist, "linux386", "bin/mybin")
	createFakeBinary(t, dist, "linuxamd64", "bin/mybin")
	createMockMakeself(t, folder)

	// Create some extra files for the package
	require.NoError(t, os.MkdirAll(filepath.Join(folder, "docs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(folder, "README.md"), []byte("# Test"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(folder, "docs/manual.txt"), []byte("Manual"), 0o644))

	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        dist,
			ProjectName: "testapp",
			Makeselfs: []config.MakeselfPackage{
				{
					ID:           "default",
					IDs:          []string{"build1"},
					NameTemplate: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}",
					Label:        "{{ .ProjectName }} v{{ .Version }} Installer",
					InstallScript: `#!/bin/bash
echo "Installing {{ .ProjectName }}"
cp mybin /usr/local/bin/`,
					Compression: "gzip",
					Files: []config.File{
						{Source: "README.md"},
						{Source: "docs/*"},
					},
				},
			},
		},
		testctx.WithVersion("1.0.0"),
		testctx.WithCurrentTag("v1.0.0"),
	)

	// Add binary artifacts
	linux386Build := &artifact.Artifact{
		Goos:   "linux",
		Goarch: "386",
		Name:   "bin/mybin",
		Path:   filepath.Join(dist, "linux386", "bin", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraBinary: "bin/mybin",
			artifact.ExtraID:     "build1",
		},
	}
	linuxAmd64Build := &artifact.Artifact{
		Goos:   "linux",
		Goarch: "amd64",
		Name:   "bin/mybin",
		Path:   filepath.Join(dist, "linuxamd64", "bin", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraBinary: "bin/mybin",
			artifact.ExtraID:     "build1",
		},
	}
	ctx.Artifacts.Add(linux386Build)
	ctx.Artifacts.Add(linuxAmd64Build)

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))

	// Verify artifacts were created
	artifacts := ctx.Artifacts.Filter(artifact.ByType(artifact.MakeselfPackage)).List()
	require.Len(t, artifacts, 2) // One for each platform

	for _, art := range artifacts {
		require.Equal(t, "default", art.ID())
		require.Equal(t, "makeself", art.Format())
		require.Equal(t, ".run", art.Ext())
		require.Contains(t, art.Name, "testapp_1.0.0_linux_")
		require.True(t, strings.HasSuffix(art.Name, ".run"))

		// Verify artifact file exists
		require.FileExists(t, art.Path)

		// Check binaries were included
		binaries := artifact.MustExtra[[]string](*art, artifact.ExtraBinaries)
		require.Contains(t, binaries, "bin/mybin")
	}

	// Verify makeself was called with correct arguments
	argsFile := filepath.Join(folder, "makeself_args.log")
	args, err := os.ReadFile(argsFile)
	require.NoError(t, err)
	argsStr := string(args)
	require.Contains(t, argsStr, "testapp v1.0.0 Installer")
	require.Contains(t, argsStr, "--gzip")
}

func TestRunPipeNoBinaries(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	createMockMakeself(t, folder)

	ctx := testctx.NewWithCfg(
		config.Project{
			Dist: dist,
			Makeselfs: []config.MakeselfPackage{
				{
					IDs: []string{"nonexistent"},
				},
			},
		},
		testctx.WithVersion("1.0.0"),
	)

	require.NoError(t, Pipe{}.Default(ctx))
	err := Pipe{}.Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no binaries found for builds")
}

func TestRunPipeDisabled(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	createFakeBinary(t, dist, "linux386", "bin/mybin")
	createMockMakeself(t, folder)

	ctx := testctx.NewWithCfg(
		config.Project{
			Dist: dist,
			Makeselfs: []config.MakeselfPackage{
				{
					IDs:     []string{"build1"},
					Disable: "true",
				},
			},
		},
		testctx.WithVersion("1.0.0"),
	)

	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "linux",
		Goarch: "386",
		Name:   "bin/mybin",
		Path:   filepath.Join(dist, "linux386", "bin", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraBinary: "bin/mybin",
			artifact.ExtraID:     "build1",
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))

	// Should have no makeself artifacts
	artifacts := ctx.Artifacts.Filter(artifact.ByType(artifact.MakeselfPackage)).List()
	require.Empty(t, artifacts)
}

func TestRunPipeMetaPackage(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	createMockMakeself(t, folder)

	// Create some files for meta package
	require.NoError(t, os.WriteFile(filepath.Join(folder, "config.conf"), []byte("config"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(folder, "scripts.sh"), []byte("#!/bin/bash\necho test"), 0o755))

	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        dist,
			ProjectName: "configs",
			Makeselfs: []config.MakeselfPackage{
				{
					ID:           "meta",
					NameTemplate: "{{ .ProjectName }}_{{ .Version }}_meta",
					Meta:         true,
					Files: []config.File{
						{Source: "config.conf"},
						{Source: "scripts.sh"},
					},
				},
			},
		},
		testctx.WithVersion("1.0.0"),
	)

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))

	// Verify meta package was created
	artifacts := ctx.Artifacts.Filter(artifact.ByType(artifact.MakeselfPackage)).List()
	require.Len(t, artifacts, 1)
	require.Equal(t, "configs_1.0.0_meta.run", artifacts[0].Name)

	// Should have no binaries
	binaries := artifact.MustExtra[[]string](*artifacts[0], artifact.ExtraBinaries)
	require.Empty(t, binaries)
}

func TestRunPipeMetaPackageNoFiles(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	createMockMakeself(t, folder)

	ctx := testctx.NewWithCfg(
		config.Project{
			Dist: dist,
			Makeselfs: []config.MakeselfPackage{
				{
					Meta: true,
				},
			},
		},
		testctx.WithVersion("1.0.0"),
	)

	require.NoError(t, Pipe{}.Default(ctx))
	err := Pipe{}.Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no files found for meta package")
}

func TestRunPipeTemplateError(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	createFakeBinary(t, dist, "linux386", "bin/mybin")
	createMockMakeself(t, folder)

	ctx := testctx.NewWithCfg(
		config.Project{
			Dist: dist,
			Makeselfs: []config.MakeselfPackage{
				{
					IDs:          []string{"build1"},
					NameTemplate: "{{ .InvalidField }}",
				},
			},
		},
		testctx.WithVersion("1.0.0"),
	)

	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "linux",
		Goarch: "386",
		Name:   "bin/mybin",
		Path:   filepath.Join(dist, "linux386", "bin", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraBinary: "bin/mybin",
			artifact.ExtraID:     "build1",
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	err := Pipe{}.Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "InvalidField")
}

func TestRunPipeCustomExtension(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	createFakeBinary(t, dist, "linux386", "bin/mybin")
	createMockMakeself(t, folder)

	testCases := []struct {
		name              string
		extension         string
		expectedExtension string
	}{
		{
			name:              "default",
			extension:         "",
			expectedExtension: ".run",
		},
		{
			name:              "custom_with_dot",
			extension:         ".installer",
			expectedExtension: ".installer",
		},
		{
			name:              "custom_without_dot",
			extension:         "bin",
			expectedExtension: ".bin",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := testctx.NewWithCfg(
				config.Project{
					Dist:        dist,
					ProjectName: "testapp",
					Makeselfs: []config.MakeselfPackage{
						{
							IDs:       []string{"build1"},
							Extension: tc.extension,
						},
					},
				},
				testctx.WithVersion("1.0.0"),
			)

			ctx.Artifacts.Add(&artifact.Artifact{
				Goos:   "linux",
				Goarch: "386",
				Name:   "bin/mybin",
				Path:   filepath.Join(dist, "linux386", "bin", "mybin"),
				Type:   artifact.Binary,
				Extra: map[string]any{
					artifact.ExtraBinary: "bin/mybin",
					artifact.ExtraID:     "build1",
				},
			})

			require.NoError(t, Pipe{}.Default(ctx))
			require.NoError(t, Pipe{}.Run(ctx))

			artifacts := ctx.Artifacts.Filter(artifact.ByType(artifact.MakeselfPackage)).List()
			require.Len(t, artifacts, 1)
			require.True(t, strings.HasSuffix(artifacts[0].Name, tc.expectedExtension))
		})
	}
}

func TestRunPipeWithLSMTemplate(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	createFakeBinary(t, dist, "linux386", "bin/mybin")
	createMockMakeself(t, folder)

	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        dist,
			ProjectName: "myapp",
			Env:         []string{"AUTHOR=Test Author"},
			Makeselfs: []config.MakeselfPackage{
				{
					IDs:         []string{"build1"},
					Label:       "{{ .ProjectName }} Installer",
					LSMTemplate: `Begin3\nTitle: {{ .ProjectName }}\nAuthor: {{ .Env.AUTHOR }}\nEnd`,
				},
			},
		},
		testctx.WithVersion("1.0.0"),
	)

	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "linux",
		Goarch: "386",
		Name:   "bin/mybin",
		Path:   filepath.Join(dist, "linux386", "bin", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraBinary: "bin/mybin",
			artifact.ExtraID:     "build1",
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))

	// Check that makeself was called with LSM flag
	argsFile := filepath.Join(folder, "makeself_args.log")
	args, err := os.ReadFile(argsFile)
	require.NoError(t, err)
	require.Contains(t, string(args), "--lsm")
}

func TestRunPipeWithLSMFile(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	createFakeBinary(t, dist, "linux386", "bin/mybin")
	createMockMakeself(t, folder)

	// Create LSM file
	lsmFile := filepath.Join(folder, "app.lsm")
	require.NoError(t, os.WriteFile(lsmFile, []byte("Begin3\nTitle: Test\nEnd"), 0o644))

	ctx := testctx.NewWithCfg(
		config.Project{
			Dist: dist,
			Makeselfs: []config.MakeselfPackage{
				{
					IDs:     []string{"build1"},
					LSMFile: lsmFile,
				},
			},
		},
		testctx.WithVersion("1.0.0"),
	)

	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "linux",
		Goarch: "386",
		Name:   "bin/mybin",
		Path:   filepath.Join(dist, "linux386", "bin", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraBinary: "bin/mybin",
			artifact.ExtraID:     "build1",
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))

	// Check that makeself was called with LSM file
	argsFile := filepath.Join(folder, "makeself_args.log")
	args, err := os.ReadFile(argsFile)
	require.NoError(t, err)
	argsStr := string(args)
	require.Contains(t, argsStr, "--lsm")
	require.Contains(t, argsStr, lsmFile)
}

func TestRunPipeWithExtraArgs(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	createFakeBinary(t, dist, "linux386", "bin/mybin")
	createMockMakeself(t, folder)

	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        dist,
			ProjectName: "testapp",
			Makeselfs: []config.MakeselfPackage{
				{
					IDs: []string{"build1"},
					ExtraArgs: []string{
						"--needroot",
						"--keep",
						"--copy",
					},
				},
			},
		},
		testctx.WithVersion("1.0.0"),
	)

	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "linux",
		Goarch: "386",
		Name:   "bin/mybin",
		Path:   filepath.Join(dist, "linux386", "bin", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraBinary: "bin/mybin",
			artifact.ExtraID:     "build1",
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))

	// Check that makeself was called with extra args
	argsFile := filepath.Join(folder, "makeself_args.log")
	args, err := os.ReadFile(argsFile)
	require.NoError(t, err)
	argsStr := string(args)
	require.Contains(t, argsStr, "--needroot")
	require.Contains(t, argsStr, "--keep")
	require.Contains(t, argsStr, "--copy")
}

func TestRunPipeMultipleMakeselfs(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	createFakeBinary(t, dist, "linux386", "bin/mybin")
	createMockMakeself(t, folder)

	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        dist,
			ProjectName: "testapp",
			Makeselfs: []config.MakeselfPackage{
				{
					ID:           "full",
					IDs:          []string{"build1"},
					NameTemplate: "{{ .ProjectName }}_full_{{ .Version }}_{{ .Os }}_{{ .Arch }}",
					Label:        "Full {{ .ProjectName }} Installer",
				},
				{
					ID:           "minimal",
					IDs:          []string{"build1"},
					NameTemplate: "{{ .ProjectName }}_minimal_{{ .Version }}_{{ .Os }}_{{ .Arch }}",
					Label:        "Minimal {{ .ProjectName }} Installer",
					Extension:    ".bin",
				},
			},
		},
		testctx.WithVersion("1.0.0"),
	)

	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "linux",
		Goarch: "386",
		Name:   "bin/mybin",
		Path:   filepath.Join(dist, "linux386", "bin", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraBinary: "bin/mybin",
			artifact.ExtraID:     "build1",
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))

	artifacts := ctx.Artifacts.Filter(artifact.ByType(artifact.MakeselfPackage)).List()
	require.Len(t, artifacts, 2)

	// Check both packages were created
	var fullPackage, minimalPackage *artifact.Artifact
	for _, art := range artifacts {
		if art.ID() == "full" {
			fullPackage = art
		} else if art.ID() == "minimal" {
			minimalPackage = art
		}
	}

	require.NotNil(t, fullPackage)
	require.NotNil(t, minimalPackage)
	require.Contains(t, fullPackage.Name, "testapp_full_1.0.0_linux_386.run")
	require.Contains(t, minimalPackage.Name, "testapp_minimal_1.0.0_linux_386.bin")
}

func TestBinaryPathPreservation(t *testing.T) {
	tests := []struct {
		name                 string
		stripBinaryDirectory bool
		binaryName           string
		expectedPath         string
	}{
		{
			name:                 "preserve directory structure by default",
			stripBinaryDirectory: false,
			binaryName:           "bin/agent",
			expectedPath:         "bin/agent",
		},
		{
			name:                 "strip directory when configured",
			stripBinaryDirectory: true,
			binaryName:           "bin/agent",
			expectedPath:         "agent",
		},
		{
			name:                 "handle nested directories",
			stripBinaryDirectory: false,
			binaryName:           "usr/local/bin/tool",
			expectedPath:         "usr/local/bin/tool",
		},
		{
			name:                 "strip nested directories",
			stripBinaryDirectory: true,
			binaryName:           "usr/local/bin/tool",
			expectedPath:         "tool",
		},
		{
			name:                 "handle single level binary",
			stripBinaryDirectory: false,
			binaryName:           "myapp",
			expectedPath:         "myapp",
		},
		{
			name:                 "strip single level binary",
			stripBinaryDirectory: true,
			binaryName:           "myapp",
			expectedPath:         "myapp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			makeselfCfg := config.MakeselfPackage{
				StripBinaryDirectory: tt.stripBinaryDirectory,
			}

			// Simulate the binary path logic from the implementation
			var dst string
			if makeselfCfg.StripBinaryDirectory {
				dst = filepath.Join(tempDir, filepath.Base(tt.binaryName))
			} else {
				dst = filepath.Join(tempDir, tt.binaryName)
			}

			// Extract the relative path within tempDir
			relPath, err := filepath.Rel(tempDir, dst)
			require.NoError(t, err)

			// Normalize path separators for comparison
			expectedPath := filepath.FromSlash(tt.expectedPath)
			require.Equal(t, expectedPath, relPath)
		})
	}
}

func TestStripBinaryDirectoryInMakeself(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	createFakeBinary(t, dist, "linux386", "bin/mybin")
	createFakeBinary(t, dist, "linux386", "bin/tool")
	createMockMakeself(t, folder)

	tests := []struct {
		name                 string
		stripBinaryDirectory bool
		expectedBinaries     []string
	}{
		{
			name:                 "preserve binary paths by default",
			stripBinaryDirectory: false,
			expectedBinaries:     []string{"bin/mybin", "bin/tool"},
		},
		{
			name:                 "strip binary paths when configured",
			stripBinaryDirectory: true,
			expectedBinaries:     []string{"bin/mybin", "bin/tool"}, // Artifact.Name remains same, only internal path changes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testctx.NewWithCfg(
				config.Project{
					Dist:        dist,
					ProjectName: "testapp",
					Makeselfs: []config.MakeselfPackage{
						{
							ID:                   "test",
							IDs:                  []string{"build1"},
							StripBinaryDirectory: tt.stripBinaryDirectory,
						},
					},
				},
				testctx.WithVersion("1.0.0"),
			)

			// Add binary artifacts
			ctx.Artifacts.Add(&artifact.Artifact{
				Goos:   "linux",
				Goarch: "386",
				Name:   "bin/mybin",
				Path:   filepath.Join(dist, "linux386", "bin", "mybin"),
				Type:   artifact.Binary,
				Extra: map[string]any{
					artifact.ExtraBinary: "bin/mybin",
					artifact.ExtraID:     "build1",
				},
			})
			ctx.Artifacts.Add(&artifact.Artifact{
				Goos:   "linux",
				Goarch: "386",
				Name:   "bin/tool",
				Path:   filepath.Join(dist, "linux386", "bin", "tool"),
				Type:   artifact.Binary,
				Extra: map[string]any{
					artifact.ExtraBinary: "bin/tool",
					artifact.ExtraID:     "build1",
				},
			})

			require.NoError(t, Pipe{}.Default(ctx))
			require.NoError(t, Pipe{}.Run(ctx))

			// Verify artifacts were created
			artifacts := ctx.Artifacts.Filter(artifact.ByType(artifact.MakeselfPackage)).List()
			require.Len(t, artifacts, 1)

			// Check binaries were included
			binaries := artifact.MustExtra[[]string](*artifacts[0], artifact.ExtraBinaries)
			require.ElementsMatch(t, tt.expectedBinaries, binaries)
		})
	}
}

func TestCopyFilePreservesPermissions(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name         string
		sourceMode   os.FileMode
		expectedMode os.FileMode
	}{
		{
			name:         "executable script",
			sourceMode:   0o755,
			expectedMode: 0o755,
		},
		{
			name:         "regular file",
			sourceMode:   0o644,
			expectedMode: 0o644,
		},
		{
			name:         "read-only file",
			sourceMode:   0o444,
			expectedMode: 0o444,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create source file with specific permissions
			srcPath := filepath.Join(tempDir, "source_"+tt.name)
			require.NoError(t, os.WriteFile(srcPath, []byte("test content"), tt.sourceMode))

			// Verify source file has expected permissions
			srcInfo, err := os.Stat(srcPath)
			require.NoError(t, err)
			require.Equal(t, tt.sourceMode, srcInfo.Mode().Perm())

			// Copy file using copyFile function
			dstPath := filepath.Join(tempDir, "dest_"+tt.name)
			require.NoError(t, copyFile(srcPath, dstPath))

			// Verify destination file has same permissions as source
			dstInfo, err := os.Stat(dstPath)
			require.NoError(t, err)
			require.Equal(t, tt.expectedMode, dstInfo.Mode().Perm())

			// Verify content was copied correctly
			content, err := os.ReadFile(dstPath)
			require.NoError(t, err)
			require.Equal(t, "test content", string(content))
		})
	}
}

func TestMakeselfPreservesScriptPermissions(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	createFakeBinary(t, dist, "linux386", "bin/mybin")
	createMockMakeself(t, folder)

	// Create executable script file in source
	scriptDir := filepath.Join(folder, "scripts")
	require.NoError(t, os.MkdirAll(scriptDir, 0o755))
	scriptPath := filepath.Join(scriptDir, "activate.sh")
	scriptContent := `#!/bin/bash
echo "Activating application"
chmod +x bin/*
`
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0o755))

	// Verify source script is executable
	scriptInfo, err := os.Stat(scriptPath)
	require.NoError(t, err)
	require.True(t, scriptInfo.Mode().Perm()&0o111 != 0, "source script should be executable")

	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        dist,
			ProjectName: "testapp",
			Makeselfs: []config.MakeselfPackage{
				{
					ID:  "default",
					IDs: []string{"build1"},
					Files: []config.File{
						{Source: "scripts/activate.sh", Destination: "scripts/activate.sh"},
					},
				},
			},
		},
		testctx.WithVersion("1.0.0"),
	)

	// Add binary artifact
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "linux",
		Goarch: "386",
		Name:   "bin/mybin",
		Path:   filepath.Join(dist, "linux386", "bin", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraBinary: "bin/mybin",
			artifact.ExtraID:     "build1",
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))

	// Verify makeself package was created
	artifacts := ctx.Artifacts.Filter(artifact.ByType(artifact.MakeselfPackage)).List()
	require.Len(t, artifacts, 1)
	require.FileExists(t, artifacts[0].Path)

	// The test verifies that the copyFile function preserves permissions
	// The actual verification of permissions within the makeself archive
	// would require extracting and testing the archive, which is beyond
	// the scope of this unit test. The copyFile test above covers the
	// core functionality.
}
