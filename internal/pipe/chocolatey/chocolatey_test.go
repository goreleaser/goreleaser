package chocolatey

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/caarlos0/log"
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

func TestPublishRedactsCredentials(t *testing.T) {
	testlib.SkipIfWindows(t, "uses a sh executable fixture")

	for _, tc := range []struct {
		name     string
		template string
		key      string
	}{
		{"default env", "{{ .Env.CHOCOLATEY_API_KEY }}", "synthetic-config-key"},
		{"custom env", "{{ .Env.CUSTOM_AUTH }}", "synthetic-custom-key"},
		{"literal", "synthetic-literal-key", "synthetic-literal-key"},
	} {
		for _, exit := range []int{0, 7} {
			t.Run(fmt.Sprintf("%s/exit%d", tc.name, exit), func(t *testing.T) {
				var logs bytes.Buffer
				previous := log.Log
				log.Log = log.New(&logs)
				log.SetLevel(log.DebugLevel)
				t.Cleanup(func() { log.Log = previous })

				dir := t.TempDir()
				script := fmt.Sprintf(`#!/bin/sh
test "$#" = 6 && test "$1" = push && test "$4" = --api-key || exit 99
test "$5" = '%s' || exit 98
test "$CHOCOLATEY_API_KEY" = synthetic-inherited-key || exit 97
printf '%%s' "$5" > "$6.received"
printf 'server diagnostic: %%s %%s %%s\n' "$5" "$CHOCOLATEY_API_KEY" "$HOST_TOKEN"
printf 'stderr diagnostic: %%s %%s\n' "$RELEASE_KEY" synthetic-context-release-key >&2
exit %d
`, tc.key, exit)
				require.NoError(t, os.WriteFile(filepath.Join(dir, "choco"), []byte(script), 0o755))
				t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
				t.Setenv("CHOCOLATEY_API_KEY", "synthetic-inherited-key")
				t.Setenv("HOST_TOKEN", "synthetic-host-token")
				t.Setenv("RELEASE_KEY", "synthetic-inherited-release-key")
				ctx := testctx.Wrap(t.Context(), testctx.WithEnv(map[string]string{
					"CHOCOLATEY_API_KEY": "synthetic-config-key",
					"CUSTOM_AUTH":        "synthetic-custom-key",
					"RELEASE_KEY":        "synthetic-context-release-key",
				}))
				path := filepath.Join(dir, "app.nupkg")
				ctx.Artifacts.Add(&artifact.Artifact{
					Type: artifact.PublishableChocolatey,
					Path: path,
					Extra: map[string]any{
						chocoConfigExtra: config.Chocolatey{
							Name:       "app",
							APIKey:     tc.template,
							SourceRepo: "https://packages.invalid/",
						},
					},
				})

				err := Pipe{}.Publish(ctx)
				if exit == 0 {
					require.NoError(t, err)
					require.Contains(t, logs.String(), "package sent")
				} else {
					ee, ok := errors.AsType[*exec.ExitError](err)
					require.True(t, ok, "expected exit error, got %v", err)
					require.Equal(t, exit, ee.ExitCode())
					require.ErrorContains(t, err, "failed to push chocolatey package")
					require.ErrorContains(t, err, "server diagnostic:")
					require.ErrorContains(t, err, "stderr diagnostic:")
					require.ErrorContains(t, err, "$HOST_TOKEN")
					require.ErrorContains(t, err, "$RELEASE_KEY")
				}
				received, readErr := os.ReadFile(path + ".received")
				require.NoError(t, readErr)
				require.Equal(t, tc.key, string(received))
				require.Contains(t, logs.String(), "push")
				require.Contains(t, logs.String(), "https://packages.invalid/")
				require.Contains(t, logs.String(), "$CHOCOLATEY_API_KEY")
				for _, secret := range []string{
					tc.key, "synthetic-inherited-key", "synthetic-config-key", "synthetic-host-token",
					"synthetic-inherited-release-key", "synthetic-context-release-key",
				} {
					require.NotContains(t, logs.String(), secret)
					if err != nil {
						require.NotContains(t, err.Error(), secret)
					}
				}
			})
		}
	}
}

func TestContinueOnError(t *testing.T) {
	require.True(t, Pipe{}.ContinueOnError())
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
	})
	t.Run("skip flag", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Chocolateys: []config.Chocolatey{
				{},
			},
		}, testctx.Skip(skips.Chocolatey))

		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Chocolateys: []config.Chocolatey{
				{},
			},
		})

		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestDefault(t *testing.T) {
	t.Parallel()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

			ctx := testctx.WrapWithCfg(t.Context(),
				config.Project{
					Dist:        folder,
					ProjectName: "run-all",
				},
				testctx.WithCurrentTag("v1.0.1"),
				testctx.WithVersion("1.0.1"))

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
	ctx := testctx.Wrap(t.Context(), testctx.WithVersion("1.12.3"))
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
	ctx := testctx.Wrap(t.Context(), testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"))
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

func Test_dataForMultipleArchivesSameArch(t *testing.T) {
	folder := t.TempDir()
	file := filepath.Join(folder, "archive")
	require.NoError(t, os.WriteFile(file, []byte("lorem ipsum"), 0o644))

	for _, goarch := range []string{"386", "amd64"} {
		t.Run(goarch, func(t *testing.T) {
			ctx := testctx.Wrap(t.Context(), testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"))
			artifacts := []*artifact.Artifact{
				{
					Name:    "client_1.0.0_windows_" + goarch + ".zip",
					Goos:    "windows",
					Goarch:  goarch,
					Goamd64: "v1",
					Path:    file,
				},
				{
					Name:    "server_1.0.0_windows_" + goarch + ".zip",
					Goos:    "windows",
					Goarch:  goarch,
					Goamd64: "v1",
					Path:    file,
				},
			}

			data, err := dataFor(ctx, client.NewMock(), config.Chocolatey{Name: "app"}, artifacts)
			require.ErrorIs(t, err, errMultipleArchives)
			require.Empty(t, data.Packages)
		})
	}
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

			ctx := testctx.Wrap(t.Context())
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
