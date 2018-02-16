package scoop

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
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
	assert.NotEmpty(t, ctx.Config.Scoop.CommitAuthor.Name)
	assert.NotEmpty(t, ctx.Config.Scoop.CommitAuthor.Email)
}

func Test_doRun(t *testing.T) {
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
					Publish: true,
				},
				&DummyClient{},
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64"},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386"},
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
					Publish: true,
				},
				&DummyClient{},
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64"},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386"},
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
					Publish: true,
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
					Publish: true,
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
					Publish: false,
				},
				&DummyClient{},
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64"},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386"},
			},
			shouldErr("--skip-publish is set"),
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
					Publish: true,
				},
				&DummyClient{},
			},
			[]artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64"},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386"},
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
					Publish: true,
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
			for _, a := range tt.artifacts {
				tt.args.ctx.Artifacts.Add(a)
			}
			tt.assertError(t, doRun(tt.args.ctx, tt.args.client))
		})
	}
}

func Test_buildManifest(t *testing.T) {
	var ctx = &context.Context{
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
			},
		},
		Publish: true,
	}
	out, err := buildManifest(ctx, &DummyClient{}, []artifact.Artifact{
		{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64"},
		{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386"},
	})
	assert.NoError(t, err)
	var golden = "testdata/test_buildmanifest.json.golden"
	if *update {
		ioutil.WriteFile(golden, out.Bytes(), 0655)
	}
	bts, err := ioutil.ReadFile(golden)
	assert.NoError(t, err)
	assert.Equal(t, string(bts), out.String())
}

func Test_getDownloadURL(t *testing.T) {
	type args struct {
		ctx       *context.Context
		githubURL string
		file      string
	}
	tests := []struct {
		name    string
		args    args
		wantURL string
	}{
		{
			"simple",
			args{&context.Context{Version: "1.0.0", Config: config.Project{Release: config.Release{GitHub: config.Repo{Owner: "user", Name: "repo"}}}}, "https://github.com", "file.tar.gz"},
			"https://github.com/user/repo/releases/download/1.0.0/file.tar.gz",
		},
		{
			"custom",
			args{&context.Context{Version: "1.0.0", Config: config.Project{Release: config.Release{GitHub: config.Repo{Owner: "user", Name: "repo"}}}}, "https://git.my.company.com", "file.tar.gz"},
			"https://git.my.company.com/user/repo/releases/download/1.0.0/file.tar.gz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL := getDownloadURL(tt.args.ctx, tt.args.githubURL, tt.args.file)
			assert.Equal(t, tt.wantURL, gotURL)
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

func (client *DummyClient) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo config.Repo, content bytes.Buffer, path string) (err error) {
	client.CreatedFile = true
	bts, _ := ioutil.ReadAll(&content)
	client.Content = string(bts)
	return
}

func (client *DummyClient) Upload(ctx *context.Context, releaseID int64, name string, file *os.File) (err error) {
	return
}
