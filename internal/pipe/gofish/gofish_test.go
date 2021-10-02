package gofish

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func createTemplateData() templateData {
	return templateData{
		Desc:     "Some desc",
		Homepage: "https://google.com",
		ReleasePackages: []releasePackage{
			{
				Arch:        "amd64",
				OS:          "darwin",
				DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_x86_64.tar.gz",
				SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c68",
			},
			{
				Arch:        "arm64",
				OS:          "darwin",
				DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_arm64.tar.gz",
				SHA256:      "1633f61598ab0791e213135923624eb342196b349490sadasdsadsadasdasdsd",
			},
			{
				Arch:        "amd64",
				OS:          "linux",
				DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_x86_64.tar.gz",
				SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
			},
			{
				Arch:        "arm",
				OS:          "linux",
				DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Arm6.tar.gz",
				SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
			},
			{
				Arch:        "arm64",
				OS:          "linux",
				DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Arm64.tar.gz",
				SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
			},
			{
				Arch:        "amd64",
				OS:          "windows",
				DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_windows_amd64.zip",
				SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
			},
		},
		Name:    "Test",
		Version: "0.1.3",
	}
}

func assertDefaultTemplateData(t *testing.T, food string) {
	t.Helper()
	require.Contains(t, food, "food =")
	require.Contains(t, food, `homepage = "https://google.com"`)
	require.Contains(t, food, `url = "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_x86_64.tar.gz"`)
	require.Contains(t, food, `sha256 = "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c68"`)
	require.Contains(t, food, `local version = "0.1.3"`)
}

func TestFullFood(t *testing.T) {
	data := createTemplateData()
	data.License = "MIT"
	food, err := doBuildFood(context.New(config.Project{
		ProjectName: "foo",
	}), data)
	require.NoError(t, err)

	golden.RequireEqualLua(t, []byte(food))
}

func TestFullFoodLinuxOnly(t *testing.T) {
	data := createTemplateData()
	for i, v := range data.ReleasePackages {
		if v.OS != "linux" {
			data.ReleasePackages[i] = releasePackage{}
		}
	}

	formulae, err := doBuildFood(context.New(config.Project{
		ProjectName: "foo",
	}), data)
	require.NoError(t, err)

	golden.RequireEqualLua(t, []byte(formulae))
}

func TestFullFoodWindowsOnly(t *testing.T) {
	data := createTemplateData()
	for i, v := range data.ReleasePackages {
		if v.OS != "windows" {
			data.ReleasePackages[i] = releasePackage{}
		}
	}
	formulae, err := doBuildFood(context.New(config.Project{
		ProjectName: "foo",
	}), data)
	require.NoError(t, err)

	golden.RequireEqualLua(t, []byte(formulae))
}

func TestFormulaeSimple(t *testing.T) {
	formulae, err := doBuildFood(context.New(config.Project{}), createTemplateData())
	require.NoError(t, err)
	assertDefaultTemplateData(t, formulae)
	require.NotContains(t, formulae, "def caveats")
	require.NotContains(t, formulae, "def plist;")
}

func TestFullPipe(t *testing.T) {
	type testcase struct {
		prepare              func(ctx *context.Context)
		expectedPublishError string
	}
	for name, tt := range map[string]testcase{
		"default": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.Rigs[0].Rig.Owner = "test"
				ctx.Config.Rigs[0].Rig.Name = "test"
				ctx.Config.Rigs[0].Homepage = "https://github.com/goreleaser"
			},
		},
		"default_gitlab": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitLab
				ctx.Config.Rigs[0].Rig.Owner = "test"
				ctx.Config.Rigs[0].Rig.Name = "test"
				ctx.Config.Rigs[0].Homepage = "https://gitlab.com/goreleaser"
			},
		},
		"invalid_commit_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Rigs[0].Rig.Owner = "test"
				ctx.Config.Rigs[0].Rig.Name = "test"
				ctx.Config.Rigs[0].CommitMessageTemplate = "{{ .Asdsa }"
			},
			expectedPublishError: `template: tmpl:1: unexpected "}" in operand`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			folder := t.TempDir()
			ctx := &context.Context{
				Git: context.GitInfo{
					CurrentTag: "v1.0.1",
				},
				Version:   "1.0.1",
				Artifacts: artifact.New(),
				Env: map[string]string{
					"FOO": "foo_is_bar",
				},
				Config: config.Project{
					Dist:        folder,
					ProjectName: name,
					Rigs: []config.GoFish{
						{
							Name: name,
							IDs: []string{
								"foo",
							},
							Description: "A run pipe test formula and FOO={{ .Env.FOO }}",
						},
					},
				},
			}
			tt.prepare(ctx)
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "bar_bin.tar.gz",
				Path:   "doesnt matter",
				Goos:   "darwin",
				Goarch: "amd64",
				Type:   artifact.UploadableArchive,
				Extra: map[string]interface{}{
					"ID":     "bar",
					"Format": "tar.gz",
				},
			})
			path := filepath.Join(folder, "bin.tar.gz")
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "bin.tar.gz",
				Path:   path,
				Goos:   "darwin",
				Goarch: "amd64",
				Type:   artifact.UploadableArchive,
				Extra: map[string]interface{}{
					"ID":     "foo",
					"Format": "tar.gz",
				},
			})

			f, err := os.Create(path)
			require.NoError(t, err)
			require.NoError(t, f.Close())
			client := &DummyClient{}
			distFile := filepath.Join(folder, name+".lua")

			require.NoError(t, runAll(ctx, client))
			if tt.expectedPublishError != "" {
				require.EqualError(t, publishAll(ctx, client), tt.expectedPublishError)
				return
			}

			require.NoError(t, publishAll(ctx, client))
			require.True(t, client.CreatedFile)
			golden.RequireEqualLua(t, []byte(client.Content))

			distBts, err := os.ReadFile(distFile)
			require.NoError(t, err)
			require.Equal(t, client.Content, string(distBts))
		})
	}
}

func TestRunPipeNameTemplate(t *testing.T) {
	folder := t.TempDir()
	ctx := &context.Context{
		Git: context.GitInfo{
			CurrentTag: "v1.0.1",
		},
		Version:   "1.0.1",
		Artifacts: artifact.New(),
		Env: map[string]string{
			"FOO_BAR": "is_bar",
		},
		Config: config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Rigs: []config.GoFish{
				{
					Name: "foo_{{ .Env.FOO_BAR }}",
					Rig: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					IDs: []string{
						"foo",
					},
				},
			},
		},
	}
	path := filepath.Join(folder, "bin.tar.gz")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "bin.tar.gz",
		Path:   path,
		Goos:   "darwin",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
		Extra: map[string]interface{}{
			"ID":     "foo",
			"Format": "tar.gz",
		},
	})

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	client := &DummyClient{}
	distFile := filepath.Join(folder, "foo_is_bar.lua")

	require.NoError(t, runAll(ctx, client))
	require.NoError(t, publishAll(ctx, client))
	require.True(t, client.CreatedFile)
	golden.RequireEqualLua(t, []byte(client.Content))
	distBts, err := os.ReadFile(distFile)
	require.NoError(t, err)
	require.Equal(t, client.Content, string(distBts))
}

func TestRunPipeMultipleGoFishWithSkip(t *testing.T) {
	folder := t.TempDir()
	ctx := &context.Context{
		Git: context.GitInfo{
			CurrentTag: "v1.0.1",
		},
		Version:   "1.0.1",
		Artifacts: artifact.New(),
		Env: map[string]string{
			"FOO_BAR": "is_bar",
		},
		Config: config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Rigs: []config.GoFish{
				{
					Name: "foo",
					Rig: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					IDs: []string{
						"foo",
					},
					SkipUpload: "true",
				},
				{
					Name: "bar",
					Rig: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					IDs: []string{
						"foo",
					},
				},
				{
					Name: "foobar",
					Rig: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					IDs: []string{
						"foo",
					},
					SkipUpload: "true",
				},
			},
		},
	}
	path := filepath.Join(folder, "bin.tar.gz")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "bin.tar.gz",
		Path:   path,
		Goos:   "darwin",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
		Extra: map[string]interface{}{
			"ID":     "foo",
			"Format": "tar.gz",
		},
	})

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	cli := &DummyClient{}
	require.NoError(t, runAll(ctx, cli))
	require.EqualError(t, publishAll(ctx, cli), `rig.skip_upload is set`)
	require.True(t, cli.CreatedFile)

	for _, food := range ctx.Config.Rigs {
		distFile := filepath.Join(folder, food.Name+".lua")
		_, err := os.Stat(distFile)
		require.NoError(t, err, "file should exist: "+distFile)
	}
}

func TestRunPipeForMultipleArmVersions(t *testing.T) {
	for name, fn := range map[string]func(ctx *context.Context){
		"multiple_armv5": func(ctx *context.Context) {
			ctx.Config.Rigs[0].Goarm = "5"
		},
		"multiple_armv6": func(ctx *context.Context) {
			ctx.Config.Rigs[0].Goarm = "6"
		},
		"multiple_armv7": func(ctx *context.Context) {
			ctx.Config.Rigs[0].Goarm = "7"
		},
	} {
		t.Run(name, func(t *testing.T) {
			folder := t.TempDir()
			ctx := &context.Context{
				TokenType: context.TokenTypeGitHub,
				Git: context.GitInfo{
					CurrentTag: "v1.0.1",
				},
				Version:   "1.0.1",
				Artifacts: artifact.New(),
				Env: map[string]string{
					"FOO": "foo_is_bar",
				},
				Config: config.Project{
					Dist:        folder,
					ProjectName: name,
					Rigs: []config.GoFish{
						{
							Name:        name,
							Description: "A run pipe test formula and FOO={{ .Env.FOO }}",
							Rig: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Homepage: "https://github.com/goreleaser",
						},
					},
					GitHubURLs: config.GitHubURLs{
						Download: "https://github.com",
					},
					Release: config.Release{
						GitHub: config.Repo{
							Owner: "test",
							Name:  "test",
						},
					},
				},
			}
			fn(ctx)
			for _, a := range []struct {
				name   string
				goos   string
				goarch string
				goarm  string
			}{
				{
					name:   "bin",
					goos:   "darwin",
					goarch: "amd64",
				},
				{
					name:   "arm64",
					goos:   "linux",
					goarch: "arm64",
				},
				{
					name:   "armv5",
					goos:   "linux",
					goarch: "arm",
					goarm:  "5",
				},
				{
					name:   "armv6",
					goos:   "linux",
					goarch: "arm",
					goarm:  "6",
				},
				{
					name:   "armv7",
					goos:   "linux",
					goarch: "arm",
					goarm:  "7",
				},
			} {
				path := filepath.Join(folder, fmt.Sprintf("%s.tar.gz", a.name))
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:   fmt.Sprintf("%s.tar.gz", a.name),
					Path:   path,
					Goos:   a.goos,
					Goarch: a.goarch,
					Goarm:  a.goarm,
					Type:   artifact.UploadableArchive,
					Extra: map[string]interface{}{
						"ID":     a.name,
						"Format": "tar.gz",
					},
				})
				f, err := os.Create(path)
				require.NoError(t, err)
				require.NoError(t, f.Close())
			}

			client := &DummyClient{}
			distFile := filepath.Join(folder, name+".lua")

			require.NoError(t, runAll(ctx, client))
			require.NoError(t, publishAll(ctx, client))
			require.True(t, client.CreatedFile)
			golden.RequireEqualLua(t, []byte(client.Content))

			distBts, err := os.ReadFile(distFile)
			require.NoError(t, err)
			require.Equal(t, client.Content, string(distBts))
		})
	}
}

func TestRunPipeNoBuilds(t *testing.T) {
	ctx := &context.Context{
		TokenType: context.TokenTypeGitHub,
		Config: config.Project{
			Rigs: []config.GoFish{
				{
					Rig: config.RepoRef{
						Owner: "test",
						Name:  "test",
					},
				},
			},
		},
	}
	client := &DummyClient{}
	require.Equal(t, ErrNoArchivesFound, runAll(ctx, client))
	require.False(t, client.CreatedFile)
}

func TestRunPipeMultipleArchivesSameOsBuild(t *testing.T) {
	ctx := context.New(
		config.Project{
			Rigs: []config.GoFish{
				{
					Rig: config.RepoRef{
						Owner: "test",
						Name:  "test",
					},
				},
			},
		},
	)

	ctx.TokenType = context.TokenTypeGitHub
	f, err := ioutil.TempFile(t.TempDir(), "")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, f.Close())
	})

	tests := []struct {
		expectedError error
		osarchs       []struct {
			goos   string
			goarch string
			goarm  string
		}
	}{
		{
			expectedError: ErrMultipleArchivesSameOS,
			osarchs: []struct {
				goos   string
				goarch string
				goarm  string
			}{
				{
					goos:   "darwin",
					goarch: "amd64",
				},
				{
					goos:   "darwin",
					goarch: "amd64",
				},
			},
		},
		{
			expectedError: ErrMultipleArchivesSameOS,
			osarchs: []struct {
				goos   string
				goarch string
				goarm  string
			}{
				{
					goos:   "linux",
					goarch: "amd64",
				},
				{
					goos:   "linux",
					goarch: "amd64",
				},
			},
		},
		{
			expectedError: ErrMultipleArchivesSameOS,
			osarchs: []struct {
				goos   string
				goarch string
				goarm  string
			}{
				{
					goos:   "linux",
					goarch: "arm64",
				},
				{
					goos:   "linux",
					goarch: "arm64",
				},
			},
		},
		{
			expectedError: ErrMultipleArchivesSameOS,
			osarchs: []struct {
				goos   string
				goarch string
				goarm  string
			}{
				{
					goos:   "linux",
					goarch: "arm",
					goarm:  "6",
				},
				{
					goos:   "linux",
					goarch: "arm",
					goarm:  "6",
				},
			},
		},
		{
			expectedError: ErrMultipleArchivesSameOS,
			osarchs: []struct {
				goos   string
				goarch string
				goarm  string
			}{
				{
					goos:   "linux",
					goarch: "arm",
					goarm:  "5",
				},
				{
					goos:   "linux",
					goarch: "arm",
					goarm:  "6",
				},
				{
					goos:   "linux",
					goarch: "arm",
					goarm:  "7",
				},
			},
		},
	}

	for _, test := range tests {
		for idx, ttt := range test.osarchs {
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   fmt.Sprintf("bin%d", idx),
				Path:   f.Name(),
				Goos:   ttt.goos,
				Goarch: ttt.goarch,
				Type:   artifact.UploadableArchive,
				Extra: map[string]interface{}{
					"ID":     fmt.Sprintf("foo%d", idx),
					"Format": "tar.gz",
				},
			})
		}
		client := &DummyClient{}
		require.Equal(t, test.expectedError, runAll(ctx, client))
		require.False(t, client.CreatedFile)
		// clean the artifacts for the next run
		ctx.Artifacts = artifact.New()
	}
}

func TestRunPipeBinaryRelease(t *testing.T) {
	ctx := context.New(
		config.Project{
			Rigs: []config.GoFish{
				{
					Rig: config.RepoRef{
						Owner: "test",
						Name:  "test",
					},
				},
			},
		},
	)
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "bin",
		Path:   "doesnt mather",
		Goos:   "darwin",
		Goarch: "amd64",
		Type:   artifact.Binary,
	})
	client := &DummyClient{}
	require.Equal(t, ErrNoArchivesFound, runAll(ctx, client))
	require.False(t, client.CreatedFile)
}

func TestRunPipeNoUpload(t *testing.T) {
	folder := t.TempDir()
	ctx := context.New(config.Project{
		Dist:        folder,
		ProjectName: "foo",
		Release:     config.Release{},
		Rigs: []config.GoFish{
			{
				Rig: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
			},
		},
	})
	ctx.TokenType = context.TokenTypeGitHub
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.1"}
	path := filepath.Join(folder, "whatever.tar.gz")
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "bin",
		Path:   path,
		Goos:   "darwin",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
		Extra: map[string]interface{}{
			"ID":     "foo",
			"Format": "tar.gz",
		},
	})
	client := &DummyClient{}

	assertNoPublish := func(t *testing.T) {
		t.Helper()
		require.NoError(t, runAll(ctx, client))
		testlib.AssertSkipped(t, publishAll(ctx, client))
		require.False(t, client.CreatedFile)
	}
	t.Run("skip upload true", func(t *testing.T) {
		ctx.Config.Rigs[0].SkipUpload = "true"
		ctx.Semver.Prerelease = ""
		assertNoPublish(t)
	})
	t.Run("skip upload auto", func(t *testing.T) {
		ctx.Config.Rigs[0].SkipUpload = "auto"
		ctx.Semver.Prerelease = "beta1"
		assertNoPublish(t)
	})
}

func TestRunEmptyTokenType(t *testing.T) {
	folder := t.TempDir()
	ctx := context.New(config.Project{
		Dist:        folder,
		ProjectName: "foo",
		Release:     config.Release{},
		Rigs: []config.GoFish{
			{
				Rig: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
			},
		},
	})
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.1"}
	path := filepath.Join(folder, "whatever.tar.gz")
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "bin",
		Path:   path,
		Goos:   "darwin",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
		Extra: map[string]interface{}{
			"ID":     "foo",
			"Format": "tar.gz",
		},
	})
	client := &DummyClient{}
	require.NoError(t, runAll(ctx, client))
}

func TestDefault(t *testing.T) {
	testlib.Mktmp(t)

	ctx := &context.Context{
		TokenType: context.TokenTypeGitHub,
		Config: config.Project{
			ProjectName: "myproject",
			Rigs: []config.GoFish{
				{},
			},
			Builds: []config.Build{
				{
					Binary: "foo",
					Goos:   []string{"linux", "darwin"},
					Goarch: []string{"386", "amd64"},
				},
				{
					Binary: "bar",
					Goos:   []string{"linux", "darwin"},
					Goarch: []string{"386", "amd64"},
					Ignore: []config.IgnoredBuild{
						{Goos: "darwin", Goarch: "amd64"},
					},
				},
				{
					Binary: "foobar",
					Goos:   []string{"linux"},
					Goarch: []string{"amd64"},
				},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, ctx.Config.ProjectName, ctx.Config.Rigs[0].Name)
	require.NotEmpty(t, ctx.Config.Rigs[0].CommitAuthor.Name)
	require.NotEmpty(t, ctx.Config.Rigs[0].CommitAuthor.Email)
	require.NotEmpty(t, ctx.Config.Rigs[0].CommitMessageTemplate)
}

func TestGHFolder(t *testing.T) {
	require.Equal(t, "bar.lua", buildFoodPath("", "bar.lua"))
	require.Equal(t, "fooo/bar.lua", buildFoodPath("fooo", "bar.lua"))
}

type DummyClient struct {
	CreatedFile bool
	Content     string
}

func (dc *DummyClient) CloseMilestone(ctx *context.Context, repo client.Repo, title string) error {
	return nil
}

func (dc *DummyClient) CreateRelease(ctx *context.Context, body string) (releaseID string, err error) {
	return
}

func (dc *DummyClient) ReleaseURLTemplate(ctx *context.Context) (string, error) {
	return "https://dummyhost/download/{{ .Tag }}/{{ .ArtifactName }}", nil
}

func (dc *DummyClient) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo client.Repo, content []byte, path, msg string) (err error) {
	dc.CreatedFile = true
	dc.Content = string(content)
	return
}

func (dc *DummyClient) Upload(ctx *context.Context, releaseID string, artifact *artifact.Artifact, file *os.File) (err error) {
	return
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			Rigs: []config.GoFish{
				{},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestRunSkipNoName(t *testing.T) {
	ctx := context.New(config.Project{
		Rigs: []config.GoFish{{}},
	})

	client := &DummyClient{}
	testlib.AssertSkipped(t, runAll(ctx, client))
}
