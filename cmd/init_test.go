package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/pipe/defaults"
	"github.com/goreleaser/goreleaser/v2/internal/static"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestInitSpecifyLanguage(t *testing.T) {
	folder := setupInitTest(t)
	cmd := newInitCmd().cmd
	config := "zigreleaser.yaml"
	cmd.SetArgs([]string{"-f", config, "-l", "zig"})
	require.NoError(t, cmd.Execute())
	require.FileExists(t, filepath.Join(folder, config))
	require.FileExists(t, filepath.Join(folder, ".gitignore"))

	bts, err := os.ReadFile(config)
	require.NoError(t, err)
	require.Equal(t, string(static.ZigExampleConfig), string(bts))
}

func TestDetectLanguage(t *testing.T) {
	for lang, expect := range map[string]struct {
		File   string
		Expect []byte
	}{
		"zig":  {"build.zig", static.ZigExampleConfig},
		"bun":  {"bun.lockb", static.BunExampleConfig},
		"rust": {"Cargo.toml", static.RustExampleConfig},
		"deno": {"deno.json", static.DenoExampleConfig},
		"uv":   {"pyproject.toml", static.UVExampleConfig},
		"go":   {"go.mod", static.GoExampleConfig}, // the file isn't actually used though, go is the default
	} {
		t.Run(expect.File, func(t *testing.T) {
			folder := setupInitTest(t)
			require.NoError(t, os.WriteFile(filepath.Join(folder, expect.File), []byte(""), 0o644))

			cmd := newInitCmd().cmd
			config := lang + "releaser.yaml"
			cmd.SetArgs([]string{"-f", config})
			require.NoError(t, cmd.Execute())
			require.FileExists(t, filepath.Join(folder, config))
			require.FileExists(t, filepath.Join(folder, ".gitignore"))

			bts, err := os.ReadFile(config)
			require.NoError(t, err)
			require.Equal(t, string(expect.Expect), string(bts))
		})
	}
}

func TestDetectLanguagePackageJSON(t *testing.T) {
	folder := setupInitTest(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "package.json"),
		[]byte(`{"devDependencies": {"@types/bun": "1.0.0"}}`),
		0o644,
	))

	cmd := newInitCmd().cmd
	config := "bunreleaser.yaml"
	cmd.SetArgs([]string{"-f", config})
	require.NoError(t, cmd.Execute())
	require.FileExists(t, filepath.Join(folder, config))
	require.FileExists(t, filepath.Join(folder, ".gitignore"))

	bts, err := os.ReadFile(config)
	require.NoError(t, err)
	require.Equal(t, string(static.BunExampleConfig), string(bts))
}

func TestDetectLanguagePyprojectTOML(t *testing.T) {
	folder := setupInitTest(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "pyproject.toml"),
		[]byte(`
[tool.poetry]
packages = [{include = "proj", from = "src"}]
`),
		0o644,
	))

	cmd := newInitCmd().cmd
	config := "poetryreleaser.yaml"
	cmd.SetArgs([]string{"-f", config})
	require.NoError(t, cmd.Execute())
	require.FileExists(t, filepath.Join(folder, config))
	require.FileExists(t, filepath.Join(folder, ".gitignore"))

	bts, err := os.ReadFile(config)
	require.NoError(t, err)
	require.Equal(t, string(static.PoetryExampleConfig), string(bts))
}

func TestInitConfigAlreadyExist(t *testing.T) {
	folder := setupInitTest(t)
	config := "foo.yaml"
	configPath := filepath.Join(folder, config)

	cmd := newInitCmd().cmd
	cmd.SetArgs([]string{"-f", config})
	content := []byte("foo: bar\n")
	require.NoError(t, os.WriteFile(configPath, content, 0o644))

	require.Error(t, cmd.Execute())
	require.FileExists(t, configPath)
	require.NoFileExists(t, filepath.Join(folder, ".gitignore"))

	bts, err := os.ReadFile(configPath)
	require.NoError(t, err)
	require.Equal(t, string(content), string(bts))
}

func TestInitGitIgnoreExists(t *testing.T) {
	folder := setupInitTest(t)
	cmd := newInitCmd().cmd

	config := "foo.yaml"
	require.NoError(t, os.WriteFile(filepath.Join(folder, ".gitignore"), []byte("mybinary\n"), 0o644))

	cmd.SetArgs([]string{"-f", config})
	require.NoError(t, cmd.Execute())
	require.FileExists(t, filepath.Join(folder, config))
	require.FileExists(t, filepath.Join(folder, ".gitignore"))

	bts, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	require.Equal(t, "mybinary\n# Added by goreleaser init:\ndist/\n", string(bts))
}

func setupInitTest(tb testing.TB) string {
	tb.Helper()

	folder := tb.TempDir()
	tb.Chdir(folder)
	return folder
}

func TestHasDistIgnored(t *testing.T) {
	t.Run("ignored", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "gitignore")
		require.NoError(t, os.WriteFile(path, []byte("foo\ndist/\nbar\n"), 0o644))
		modified, err := setupGitignore(path, []string{"dist/"})
		require.NoError(t, err)
		require.False(t, modified)
	})

	t.Run("not ignored", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "gitignore")
		require.NoError(t, os.WriteFile(path, []byte("foo\nbar\n"), 0o644))
		modified, err := setupGitignore(path, []string{"dist/", "target/"})
		require.NoError(t, err)
		require.True(t, modified)

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Contains(t, string(content), "# Added by goreleaser init:\ndist/\ntarget/\n")
	})

	t.Run("file does not exist", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "gitignore")
		modified, err := setupGitignore(path, []string{"dist/", "target/"})
		require.NoError(t, err)
		require.True(t, modified)

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Contains(t, string(content), "# Added by goreleaser init:\ndist/\ntarget/\n")
	})
}

func checkExample(t *testing.T, exampleConfig []byte) {
	t.Helper()
	cfg, err := config.LoadReader(bytes.NewReader(exampleConfig))
	require.NoError(t, err)
	ctx := testctx.WrapWithCfg(t.Context(), cfg)
	err = defaults.Pipe{}.Run(ctx)
	require.NoError(t, err)
	require.False(t, ctx.Deprecated)
}

func TestInitExampleConfigsAreNotDeprecated(t *testing.T) {
	checkExample(t, static.GoExampleConfig)
	checkExample(t, static.ZigExampleConfig)
	checkExample(t, static.BunExampleConfig)
	checkExample(t, static.DenoExampleConfig)
	checkExample(t, static.RustExampleConfig)
}

func TestSetupGitignore(t *testing.T) {
	tests := []struct {
		name           string
		existing       string
		lines          []string
		expectContent  string
		expectModified bool
	}{
		{
			name:           "empty file",
			existing:       "",
			lines:          []string{"dist/"},
			expectContent:  "# Added by goreleaser init:\ndist/\n",
			expectModified: true,
		},
		{
			name:           "no newline at end",
			existing:       "foo",
			lines:          []string{"dist/"},
			expectContent:  "foo\n# Added by goreleaser init:\ndist/\n",
			expectModified: true,
		},
		{
			name:           "no newline at end with CRLF",
			existing:       "foo\r\nbar",
			lines:          []string{"dist/"},
			expectContent:  "foo\r\nbar\n# Added by goreleaser init:\ndist/\n",
			expectModified: true,
		},
		{
			name:           "file already contains line",
			existing:       "dist/\n",
			lines:          []string{"dist/"},
			expectContent:  "dist/\n",
			expectModified: false,
		},
		{
			name:           "multiple lines",
			existing:       "",
			lines:          []string{"dist/", "target/", "build/"},
			expectContent:  "# Added by goreleaser init:\ndist/\ntarget/\nbuild/\n",
			expectModified: true,
		},
		{
			name:           "partial existing lines",
			existing:       "dist/\n",
			lines:          []string{"dist/", "target/", "build/"},
			expectContent:  "dist/\n# Added by goreleaser init:\ntarget/\nbuild/\n",
			expectModified: true,
		},
		{
			name:           "no newline at end with multiple lines",
			existing:       "foo",
			lines:          []string{"dist/", "target/", "build/"},
			expectContent:  "foo\n# Added by goreleaser init:\ndist/\ntarget/\nbuild/\n",
			expectModified: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "gitignore")
			if tt.existing != "" {
				require.NoError(t, os.WriteFile(path, []byte(tt.existing), 0o644))
			}

			modified, err := setupGitignore(path, tt.lines)
			require.NoError(t, err)
			require.Equal(t, tt.expectModified, modified)

			content, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, tt.expectContent, string(content))
		})
	}

	t.Run("write error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "gitignore")
		require.NoError(t, os.WriteFile(path, []byte(""), 0o444))

		_, err := setupGitignore(path, []string{"dist/"})
		require.Error(t, err)
	})

	t.Run("read error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "gitignore")
		require.NoError(t, os.WriteFile(path, []byte(""), 0o444))
		require.NoError(t, os.Chmod(path, 0o000))

		_, err := setupGitignore(path, []string{"dist/"})
		require.Error(t, err)
	})

	t.Run("write newline error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "gitignore")
		require.NoError(t, os.WriteFile(path, []byte("foo"), 0o444))

		_, err := setupGitignore(path, []string{"dist/"})
		require.Error(t, err)
	})
}
