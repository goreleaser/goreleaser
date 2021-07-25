package exec

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestExecute(t *testing.T) {
	ctx := context.New(config.Project{
		ProjectName: "blah",
		Archives: []config.Archive{
			{
				Replacements: map[string]string{
					"linux": "Linux",
				},
			},
		},
	})
	ctx.Env["TEST_A_SECRET"] = "x"
	ctx.Env["TEST_A_USERNAME"] = "u2"
	ctx.Version = "2.1.0"

	// Preload artifacts
	ctx.Artifacts = artifact.New()
	folder := t.TempDir()
	for _, a := range []struct {
		id  string
		ext string
		typ artifact.Type
	}{
		{"docker", "---", artifact.DockerImage},
		{"debpkg", "deb", artifact.LinuxPackage},
		{"binary", "bin", artifact.Binary},
		{"archive", "tar", artifact.UploadableArchive},
		{"ubinary", "ubi", artifact.UploadableBinary},
		{"checksum", "sum", artifact.Checksum},
		{"signature", "sig", artifact.Signature},
	} {
		file := filepath.Join(folder, "a."+a.ext)
		require.NoError(t, os.WriteFile(file, []byte("lorem ipsum"), 0o644))
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   "a." + a.ext,
			Goos:   "linux",
			Goarch: "amd64",
			Path:   file,
			Type:   a.typ,
			Extra: map[string]interface{}{
				"ID": a.id,
			},
		})
	}

	osEnv := func(ignores ...string) []string {
		var result []string
	outer:
		for _, key := range passthroughEnvVars {
			for _, ignore := range ignores {
				if key == ignore {
					continue outer
				}
			}
			if value := os.Getenv(key); value != "" {
				result = append(result, key+"="+value)
			}
		}
		return result
	}

	testCases := []struct {
		name       string
		publishers []config.Publisher
		expectErr  error
	}{
		{
			"filter by IDs",
			[]config.Publisher{
				{
					Name: "test",
					IDs:  []string{"archive"},
					Cmd:  MockCmd + " {{ .ArtifactName }}",
					Env: []string{
						MarshalMockEnv(&MockData{
							AnyOf: []MockCall{
								{ExpectedArgs: []string{"a.tar"}, ExitCode: 0, ExpectedEnv: osEnv()},
							},
						}),
					},
				},
			},
			nil,
		},
		{
			"no filter",
			[]config.Publisher{
				{
					Name: "test",
					Cmd:  MockCmd + " {{ .ArtifactName }}",
					Env: []string{
						MarshalMockEnv(&MockData{
							AnyOf: []MockCall{
								{ExpectedArgs: []string{"a.deb"}, ExitCode: 0, ExpectedEnv: osEnv()},
								{ExpectedArgs: []string{"a.ubi"}, ExitCode: 0, ExpectedEnv: osEnv()},
								{ExpectedArgs: []string{"a.tar"}, ExitCode: 0, ExpectedEnv: osEnv()},
							},
						}),
					},
				},
			},
			nil,
		},
		{
			"include checksum",
			[]config.Publisher{
				{
					Name:     "test",
					Checksum: true,
					Cmd:      MockCmd + " {{ .ArtifactName }}",
					Env: []string{
						MarshalMockEnv(&MockData{
							AnyOf: []MockCall{
								{ExpectedArgs: []string{"a.deb"}, ExitCode: 0, ExpectedEnv: osEnv()},
								{ExpectedArgs: []string{"a.ubi"}, ExitCode: 0, ExpectedEnv: osEnv()},
								{ExpectedArgs: []string{"a.tar"}, ExitCode: 0, ExpectedEnv: osEnv()},
								{ExpectedArgs: []string{"a.sum"}, ExitCode: 0, ExpectedEnv: osEnv()},
							},
						}),
					},
				},
			},
			nil,
		},
		{
			"include signatures",
			[]config.Publisher{
				{
					Name:      "test",
					Signature: true,
					Cmd:       MockCmd + " {{ .ArtifactName }}",
					Env: []string{
						MarshalMockEnv(&MockData{
							AnyOf: []MockCall{
								{ExpectedArgs: []string{"a.deb"}, ExitCode: 0, ExpectedEnv: osEnv()},
								{ExpectedArgs: []string{"a.ubi"}, ExitCode: 0, ExpectedEnv: osEnv()},
								{ExpectedArgs: []string{"a.tar"}, ExitCode: 0, ExpectedEnv: osEnv()},
								{ExpectedArgs: []string{"a.sig"}, ExitCode: 0, ExpectedEnv: osEnv()},
							},
						}),
					},
				},
			},
			nil,
		},
		{
			"try dir templating",
			[]config.Publisher{
				{
					Name:      "test",
					Signature: true,
					IDs:       []string{"debpkg"},
					Dir:       "{{ dir .ArtifactPath }}",
					Cmd:       MockCmd + " {{ .ArtifactName }}",
					Env: []string{
						MarshalMockEnv(&MockData{
							AnyOf: []MockCall{
								{ExpectedArgs: []string{"a.deb"}, ExitCode: 0, ExpectedEnv: osEnv()},
							},
						}),
					},
				},
			},
			nil,
		},
		{
			"check env templating",
			[]config.Publisher{
				{
					Name: "test",
					IDs:  []string{"debpkg"},
					Cmd:  MockCmd,
					Env: []string{
						"PROJECT={{.ProjectName}}",
						"ARTIFACT={{.ArtifactName}}",
						"SECRET={{.Env.TEST_A_SECRET}}",
						MarshalMockEnv(&MockData{
							AnyOf: []MockCall{
								{
									ExpectedEnv: append(
										[]string{"PROJECT=blah", "ARTIFACT=a.deb", "SECRET=x"},
										osEnv()...,
									),
									ExitCode: 0,
								},
							},
						}),
					},
				},
			},
			nil,
		},
		{
			"override path",
			[]config.Publisher{
				{
					Name: "test",
					IDs:  []string{"debpkg"},
					Cmd:  MockCmd,
					Env: []string{
						"PATH=/something-else",
						MarshalMockEnv(&MockData{
							AnyOf: []MockCall{
								{
									ExpectedEnv: append(
										[]string{"PATH=/something-else"},
										osEnv("PATH")...,
									),
									ExitCode: 0,
								},
							},
						}),
					},
				},
			},
			nil,
		},
		{
			"command error",
			[]config.Publisher{
				{
					Name: "test",
					IDs:  []string{"debpkg"},
					Cmd:  MockCmd + " {{.ArtifactName}}",
					Env: []string{
						MarshalMockEnv(&MockData{
							AnyOf: []MockCall{
								{
									ExpectedArgs: []string{"a.deb"},
									ExpectedEnv:  osEnv(),
									Stderr:       "test error",
									ExitCode:     1,
								},
							},
						}),
					},
				},
			},
			// stderr is sent to output via logger
			fmt.Errorf(`publishing: %s failed: exit status 1`, MockCmd),
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {
			err := Execute(ctx, tc.publishers)
			if tc.expectErr == nil {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Equal(t, tc.expectErr.Error(), err.Error())
		})
	}
}
