package scoop

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update .golden files")

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestDefault(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()

	var ctx = &context.Context{
		Config: config.Project{
			ProjectName: "barr",
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
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, ctx.Config.ProjectName, ctx.Config.Scoop.Name)
	assert.NotEmpty(t, ctx.Config.Scoop.CommitAuthor.Name)
	assert.NotEmpty(t, ctx.Config.Scoop.CommitAuthor.Email)
}

func Test_doRun(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	var file = filepath.Join(folder, "archive")
	require.NoError(t, ioutil.WriteFile(file, []byte("lorem ipsum"), 0644))

	type errChecker func(*testing.T, error)
	var shouldErr = func(msg string) errChecker {
		return func(t *testing.T, err error) {
			assert.Error(t, err)
			assert.EqualError(t, err, msg)
		}
	}
	var shouldNotErr = func(t *testing.T, err error) {
		assert.NoError(t, err)
	}
	type args struct {
		ctx    *context.Context
		client client.Client
	}
	tests := []struct {
		name        string
		args        args
		artifacts   []artifact.Artifact
		assertError errChecker
	}{
		{
			"valid",
			args{
				&context.Context{
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archive: config.Archive{
							Format: "tar.gz",
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							Bucket: config.Repo{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldNotErr,
		},
		{
			"valid",
			args{
				&context.Context{
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						GitHubURLs: config.GitHubURLs{Download: "https://api.custom.github.enterprise.com"},
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archive: config.Archive{
							Format: "tar.gz",
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							Bucket: config.Repo{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldNotErr,
		},
		{
			"no windows build",
			args{
				&context.Context{
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test"},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archive: config.Archive{
							Format: "tar.gz",
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							Bucket: config.Repo{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1_linux_amd64.tar.gz", Goos: "linux", Goarch: "amd64"},
				{Name: "foo_1.0.1_linux_386.tar.gz", Goos: "linux", Goarch: "386"},
			},
			shouldErr("scoop requires a windows build"),
		},
		{
			"no scoop",
			args{
				&context.Context{
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archive: config.Archive{
							Format: "tar.gz",
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
					},
				},
				&DummyClient{},
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64"},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386"},
			},
			shouldErr("scoop section is not configured"),
		},
		{
			"no publish",
			args{
				&context.Context{
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archive: config.Archive{
							Format: "tar.gz",
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							Bucket: config.Repo{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
					SkipPublish: true,
				},
				&DummyClient{},
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldErr(pipe.ErrSkipPublishEnabled.Error()),
		},
		{
			"is draft",
			args{
				&context.Context{
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archive: config.Archive{
							Format: "tar.gz",
						},
						Release: config.Release{
							Draft: true,
						},
						Scoop: config.Scoop{
							Bucket: config.Repo{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldErr("release is marked as draft"),
		},
		{
			"no archive",
			args{
				&context.Context{
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archive: config.Archive{
							Format: "binary",
						},
						Release: config.Release{
							Draft: true,
						},
						Scoop: config.Scoop{
							Bucket: config.Repo{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64"},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386"},
			},
			shouldErr("archive format is binary"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx = tt.args.ctx
			for _, a := range tt.artifacts {
				ctx.Artifacts.Add(a)
			}
			require.NoError(t, Pipe{}.Default(ctx))
			tt.assertError(t, doRun(ctx, tt.args.client))
		})
	}
}

func Test_buildManifest(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	require.NoError(t, err)
	var file = filepath.Join(folder, "archive")
	require.NoError(t, ioutil.WriteFile(file, []byte("lorem ipsum"), 0644))

	tests := []struct {
		filename string
		ctx      *context.Context
	}{
		{
			"testdata/test_buildmanifest.json.golden",
			&context.Context{
				Git: context.GitInfo{
					CurrentTag: "v1.0.1",
				},
				Version:   "1.0.1",
				Artifacts: artifact.New(),
				Config: config.Project{
					GitHubURLs: config.GitHubURLs{
						Download: "https://github.com",
					},
					Dist:        ".",
					ProjectName: "run-pipe",
					Archive: config.Archive{
						Format: "tar.gz",
					},
					Release: config.Release{
						GitHub: config.Repo{
							Owner: "test",
							Name:  "test",
						},
					},
					Scoop: config.Scoop{
						Bucket: config.Repo{
							Owner: "test",
							Name:  "test",
						},
						Description: "A run pipe test formula",
						Homepage:    "https://github.com/goreleaser",
						Persist:     []string{"data", "config", "test.ini"},
					},
				},
			},
		},
		{
			"testdata/test_buildmanifest_url_template.json.golden",
			&context.Context{
				Git: context.GitInfo{
					CurrentTag: "v1.0.1",
				},
				Version:   "1.0.1",
				Artifacts: artifact.New(),
				Config: config.Project{
					GitHubURLs: config.GitHubURLs{
						Download: "https://github.com",
					},
					Builds: []config.Build{
						{Binary: "test"},
					},
					Dist:        ".",
					ProjectName: "run-pipe",
					Archive: config.Archive{
						Format: "tar.gz",
					},
					Release: config.Release{
						GitHub: config.Repo{
							Owner: "test",
							Name:  "test",
						},
					},
					Scoop: config.Scoop{
						Bucket: config.Repo{
							Owner: "test",
							Name:  "test",
						},
						Description: "A run pipe test formula",
						Homepage:    "https://github.com/goreleaser",
						URLTemplate: "http://github.mycompany.com/foo/bar/{{ .Tag }}/{{ .ArtifactName }}",
						Persist:     []string{"data.cfg", "etc"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			var ctx = tt.ctx
			err := Pipe{}.Default(ctx)
			require.NoError(t, err)
			out, err := buildManifest(ctx, []artifact.Artifact{
				{
					Name:   "foo_1.0.1_windows_amd64.tar.gz",
					Goos:   "windows",
					Goarch: "amd64",
					Path:   file,
					Extra: map[string]interface{}{
						"Builds": []artifact.Artifact{
							{
								Extra: map[string]interface{}{
									"Binary": "foo",
								},
							},
							{
								Extra: map[string]interface{}{
									"Binary": "bar",
								},
							},
						},
					},
				},
				{
					Name:   "foo_1.0.1_windows_386.tar.gz",
					Goos:   "windows",
					Goarch: "386",
					Path:   file,
					Extra: map[string]interface{}{
						"Builds": []artifact.Artifact{
							{
								Extra: map[string]interface{}{
									"Binary": "foo",
								},
							},
							{
								Extra: map[string]interface{}{
									"Binary": "bar",
								},
							},
						},
					},
				},
			})

			require.NoError(t, err)

			if *update {
				require.NoError(t, ioutil.WriteFile(tt.filename, out.Bytes(), 0655))
			}
			bts, err := ioutil.ReadFile(tt.filename)
			require.NoError(t, err)
			require.Equal(t, string(bts), out.String())
		})
	}
}

type DummyClient struct {
	CreatedFile bool
	Content     string
}

func (client *DummyClient) CreateRelease(ctx *context.Context, body string) (releaseID int64, err error) {
	return
}

func (client *DummyClient) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo config.Repo, content []byte, path, msg string) (err error) {
	client.CreatedFile = true
	client.Content = string(content)
	return
}

func (client *DummyClient) Upload(ctx *context.Context, releaseID int64, name string, file *os.File) (err error) {
	return
}
