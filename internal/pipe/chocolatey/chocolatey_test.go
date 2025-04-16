package chocolatey

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/golden"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestContinueOnError(t *testing.T) {
	require.True(t, Pipe{}.ContinueOnError())
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.New()))
	})
	t.Run("skip flag", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Chocolateys: []config.Chocolatey{
				{},
			},
		}, testctx.Skip(skips.Chocolatey))
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Chocolateys: []config.Chocolatey{
				{},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestDefault(t *testing.T) {
	testlib.Mktmp(t)

	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "myproject",
		Chocolateys: []config.Chocolatey{
			{},
		},
	}, testctx.GitHubTokenType)

	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, ctx.Config.ProjectName, ctx.Config.Chocolateys[0].Name)
	require.Equal(t, ctx.Config.ProjectName, ctx.Config.Chocolateys[0].Title)
	require.Equal(t, "v1", ctx.Config.Chocolateys[0].Goamd64)
}

func Test_doRun(t *testing.T) {
	folder := t.TempDir()
	file := filepath.Join(folder, "archive")
	require.NoError(t, os.WriteFile(file, []byte("lorem ipsum"), 0o644))

	tests := []struct {
		name      string
		choco     config.Chocolatey
		exec      func(cmd string, args ...string) ([]byte, error)
		published int
		err       string
	}{
		{
			name: "no artifacts",
			choco: config.Chocolatey{
				Name:    "app",
				IDs:     []string{"no-app"},
				Goamd64: "v1",
			},
			err: errNoWindowsArchive.Error(),
		},
		{
			name: "choco command not found",
			choco: config.Chocolatey{
				Name:    "app",
				Goamd64: "v1",
			},
			exec: func(_ string, _ ...string) ([]byte, error) {
				return nil, errors.New(`exec: "choco.exe": executable file not found in $PATH`)
			},
			err: `failed to generate chocolatey package: exec: "choco.exe": executable file not found in $PATH: `,
		},
		{
			name: "skip publish",
			choco: config.Chocolatey{
				Name:        "app",
				Goamd64:     "v1",
				SkipPublish: true,
			},
			exec: func(cmd string, args ...string) ([]byte, error) {
				checkPackCmd(t, cmd, args...)
				return []byte("success"), nil
			},
		},
		{
			name: "success",
			choco: config.Chocolatey{
				Name:    "app",
				Goamd64: "v1",
			},
			exec: func(cmd string, args ...string) ([]byte, error) {
				checkPackCmd(t, cmd, args...)
				return []byte("success"), nil
			},
			published: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd = fakeCmd{execFn: tt.exec}
			t.Cleanup(func() {
				cmd = stdCmd{}
			})

			ctx := testctx.NewWithCfg(
				config.Project{
					Dist:        folder,
					ProjectName: "run-all",
				},
				testctx.WithCurrentTag("v1.0.1"),
				testctx.WithVersion("1.0.1"),
			)

			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    "app_1.0.1_windows_amd64.zip",
				Path:    file,
				Goos:    "windows",
				Goarch:  "amd64",
				Goamd64: "v1",
				Type:    artifact.UploadableArchive,
				Extra: map[string]any{
					artifact.ExtraID:     "app",
					artifact.ExtraFormat: "zip",
				},
			})

			client := client.NewMock()
			got := doRun(ctx, client, tt.choco)

			var err string
			if got != nil {
				err = got.Error()
			}
			if tt.err != err {
				t.Errorf("Unexpected error: %s (expected %s)", err, tt.err)
			}

			list := ctx.Artifacts.Filter(artifact.ByType(artifact.PublishableChocolatey)).List()
			require.Len(t, list, tt.published)
		})
	}
}

func Test_buildNuspec(t *testing.T) {
	ctx := testctx.New(testctx.WithVersion("1.12.3"))
	choco := config.Chocolatey{
		Name:        "goreleaser",
		IDs:         []string{},
		Title:       "GoReleaser",
		Authors:     "caarlos0",
		ProjectURL:  "https://goreleaser.com/",
		Tags:        "go docker homebrew golang package",
		Summary:     "Deliver Go binaries as fast and easily as possible",
		Description: "GoReleaser builds Go binaries for several platforms, creates a GitHub release and then pushes a Homebrew formula to a tap repository. All that wrapped in your favorite CI.",
		Dependencies: []config.ChocolateyDependency{
			{ID: "nfpm"},
		},
	}

	out, err := buildNuspec(ctx, choco)
	require.NoError(t, err)

	golden.RequireEqualExt(t, out, ".nuspec")
}

func Test_buildTemplate(t *testing.T) {
	folder := t.TempDir()
	file := filepath.Join(folder, "archive")
	require.NoError(t, os.WriteFile(file, []byte("lorem ipsum"), 0o644))
	ctx := testctx.New(testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"))
	artifacts := []*artifact.Artifact{
		{
			Name:    "app_1.0.0_windows_386.zip",
			Goos:    "windows",
			Goarch:  "386",
			Goamd64: "v1",
			Path:    file,
		},
		{
			Name:    "app_1.0.0_windows_amd64.zip",
			Goos:    "windows",
			Goarch:  "amd64",
			Goamd64: "v1",
			Path:    file,
		},
	}

	choco := config.Chocolatey{
		Name: "app",
	}

	client := client.NewMock()

	data, err := dataFor(ctx, client, choco, artifacts)
	if err != nil {
		t.Error(err)
	}

	out, err := buildTemplate(choco.Name, scriptTemplate, data)
	require.NoError(t, err)

	golden.RequireEqualExt(t, out, ".script.ps1")
}

func TestPublish(t *testing.T) {
	folder := t.TempDir()
	file := filepath.Join(folder, "archive")
	require.NoError(t, os.WriteFile(file, []byte("lorem ipsum"), 0o644))

	fakenu := filepath.Join(t.TempDir(), "foo.nupkg")
	require.NoError(t, os.WriteFile(fakenu, []byte("fake nupkg"), 0o644))

	tests := []struct {
		name      string
		artifacts []artifact.Artifact
		exec      func(cmd string, args ...string) ([]byte, error)
		skip      bool
		err       string
	}{
		{
			name: "no artifacts",
		},
		{
			name: "no api key",
			artifacts: []artifact.Artifact{
				{
					Type: artifact.PublishableChocolatey,
					Name: "app.1.0.1.nupkg",
					Extra: map[string]any{
						artifact.ExtraFormat: nupkgFormat,
						chocoConfigExtra:     config.Chocolatey{},
					},
				},
			},
		},
		{
			name: "push error",
			artifacts: []artifact.Artifact{
				{
					Type: artifact.PublishableChocolatey,
					Name: "app.1.0.1.nupkg",
					Extra: map[string]any{
						artifact.ExtraFormat: nupkgFormat,
						chocoConfigExtra: config.Chocolatey{
							APIKey: "abcd",
						},
					},
				},
			},
			exec: func(_ string, _ ...string) ([]byte, error) {
				return nil, errors.New(`unable to push`)
			},
			err: "failed to push chocolatey package: unable to push: ",
		},
		{
			name: "success",
			artifacts: []artifact.Artifact{
				{
					Type: artifact.PublishableChocolatey,
					Name: "app.1.0.1.nupkg",
					Path: fakenu,
					Extra: map[string]any{
						artifact.ExtraFormat: nupkgFormat,
						chocoConfigExtra: config.Chocolatey{
							APIKey:     "abcd",
							SourceRepo: "abc",
						},
					},
				},
			},
			exec: func(cmd string, args ...string) ([]byte, error) {
				checkPushCmd(t, cmd, args...)
				return []byte("success"), nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd = fakeCmd{execFn: tt.exec}
			t.Cleanup(func() {
				cmd = stdCmd{}
			})

			ctx := testctx.New()
			for _, artifact := range tt.artifacts {
				ctx.Artifacts.Add(&artifact)
			}

			got := Pipe{}.Publish(ctx)

			var err string
			if got != nil {
				err = got.Error()
			}
			if tt.err != err {
				t.Errorf("Unexpected error: %s (expected %s)", err, tt.err)
			}
		})
	}
}

func TestDependencies(t *testing.T) {
	require.Equal(t, []string{"choco"}, Pipe{}.Dependencies(nil))
}

type fakeCmd struct {
	execFn func(cmd string, args ...string) ([]byte, error)
}

var _ cmder = fakeCmd{}

func (f fakeCmd) Exec(_ *context.Context, cmd string, args ...string) ([]byte, error) {
	return f.execFn(cmd, args...)
}

func checkPushCmd(tb testing.TB, cmd string, args ...string) {
	tb.Helper()
	tb.Log("would have run:", cmd, args)
	require.Len(tb, args, 6)
	require.Equal(tb, "choco", cmd)
	require.FileExists(tb, args[5])
}

func checkPackCmd(tb testing.TB, cmd string, args ...string) {
	tb.Helper()
	tb.Log("would have run:", cmd, args)
	require.Len(tb, args, 4)
	require.Equal(tb, "choco", cmd)
	require.FileExists(tb, args[1])
	require.DirExists(tb, args[3])
}
