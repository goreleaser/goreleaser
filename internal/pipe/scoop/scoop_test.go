package scoop

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"runtime"
	"testing"

	"path/filepath"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: remove the Build sections as it is not really needed (or shouldn't be at least).

var update = flag.Bool("update", false, "update .golden files")

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestDefault(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()

	var ctx = &context.Context{
		Parallelism: runtime.NumCPU(),
		Config: config.Project{
			Scoops: []config.Scoop{
				{
					Description: "asd",
				},
			},
			ProjectName: "barr",
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.NotEmpty(t, ctx.Config.Scoops[0].CommitAuthor.Name)
	assert.NotEmpty(t, ctx.Config.Scoops[0].CommitAuthor.Email)
	assert.Equal(t, ctx.Config.ProjectName, ctx.Config.Scoops[0].Name)
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
					Parallelism: runtime.NumCPU(),
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Dist:        ".",
						ProjectName: "run-pipe",
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoops: []config.Scoop{
							{
								Bucket: config.Repo{
									Owner: "test",
									Name:  "test",
								},
								Description: "A run pipe test formula",
								Homepage:    "https://github.com/goreleaser",
							},
						},
					},
				},
				&DummyClient{},
			},
			[]artifact.Artifact{
				{
					Name:   "foo_1.0.1_windows_amd64.tar.gz",
					Goos:   "windows",
					Goarch: "amd64",
					Path:   file,
					Extra: map[string]string{
						"Binaries": "test",
					},
				},
				{
					Name:   "foo_1.0.1_windows_386.tar.gz",
					Goos:   "windows",
					Goarch: "386",
					Path:   file,
					Extra: map[string]string{
						"Binaries": "test",
					},
				},
			},
			shouldNotErr,
		},
		{
			"valid",
			args{
				&context.Context{
					Parallelism: runtime.NumCPU(),
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						GitHubURLs:  config.GitHubURLs{Download: "https://api.custom.github.enterprise.com"},
						Dist:        ".",
						ProjectName: "run-pipe",
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoops: []config.Scoop{
							{
								Bucket: config.Repo{
									Owner: "test",
									Name:  "test",
								},
								Description: "A run pipe test formula",
								Homepage:    "https://github.com/goreleaser",
							},
						},
					},
				},
				&DummyClient{},
			},
			[]artifact.Artifact{
				{
					Name:   "foo_1.0.1_windows_amd64.tar.gz",
					Goos:   "windows",
					Goarch: "amd64",
					Path:   file,
					Extra: map[string]string{
						"Binaries": "test",
					},
				},
				{
					Name:   "foo_1.0.1_windows_386.tar.gz",
					Goos:   "windows",
					Goarch: "386",
					Path:   file,
					Extra: map[string]string{
						"Binaries": "test",
					},
				},
			},
			shouldNotErr,
		},
		{
			"valid multiple binaries",
			args{
				&context.Context{
					Parallelism: runtime.NumCPU(),
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Dist:        ".",
						ProjectName: "multiplebinaries",
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoops: []config.Scoop{
							{
								Bucket: config.Repo{
									Owner: "test",
									Name:  "test",
								},
								Description: "A run pipe test formula",
								Homepage:    "https://github.com/goreleaser",
							},
						},
					},
				},
				&DummyClient{},
			},
			[]artifact.Artifact{
				{
					Name:   "foo_1.0.1_windows_amd64.tar.gz",
					Goos:   "windows",
					Goarch: "amd64",
					Path:   file,
					Extra: map[string]string{
						"Binaries": "test,foo,bar",
					},
				},
				{
					Name:   "foo_1.0.1_windows_386.tar.gz",
					Goos:   "windows",
					Goarch: "386",
					Path:   file,
					Extra: map[string]string{
						"Binaries": "test,foo,bar",
					},
				},
			},
			shouldNotErr,
		},
		{
			"no windows build",
			args{
				&context.Context{
					Parallelism: runtime.NumCPU(),
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Dist:        ".",
						ProjectName: "run-pipe",
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoops: []config.Scoop{
							{
								Bucket: config.Repo{
									Owner: "test",
									Name:  "test",
								},
								Description: "A run pipe test formula",
								Homepage:    "https://github.com/goreleaser",
							},
						},
					},
				},
				&DummyClient{},
			},
			[]artifact.Artifact{
				{
					Name:   "foo_1.0.1_linux_amd64.tar.gz",
					Goos:   "linux",
					Goarch: "amd64",
					Extra: map[string]string{
						"Binaries": "test",
					},
				},
				{
					Name:   "foo_1.0.1_linux_386.tar.gz",
					Goos:   "linux",
					Goarch: "386",
					Extra: map[string]string{
						"Binaries": "test",
					},
				},
			},
			shouldErr(ErrNoWindows.Error()),
		},
		{
			"no scoop",
			args{
				&context.Context{
					Parallelism: runtime.NumCPU(),
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Dist:        ".",
						ProjectName: "run-pipe",
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
				{
					Name:   "foo_1.0.1_windows_amd64.tar.gz",
					Goos:   "windows",
					Goarch: "amd64",
					Extra: map[string]string{
						"Binaries": "test",
					},
				},
				{
					Name:   "foo_1.0.1_windows_386.tar.gz",
					Goos:   "windows",
					Goarch: "386",
					Extra: map[string]string{
						"Binaries": "test",
					},
				},
			},
			shouldErr("scoop section is not configured"),
		},
		{
			"no publish",
			args{
				&context.Context{
					Parallelism: runtime.NumCPU(),
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Dist:        ".",
						ProjectName: "run-pipe",
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoops: []config.Scoop{
							{
								Bucket: config.Repo{
									Owner: "test",
									Name:  "test",
								},
								Description: "A run pipe test formula",
								Homepage:    "https://github.com/goreleaser",
							},
						},
					},
					SkipPublish: true,
				},
				&DummyClient{},
			},
			[]artifact.Artifact{
				{
					Name:   "foo_1.0.1_windows_amd64.tar.gz",
					Goos:   "windows",
					Goarch: "amd64",
					Path:   file,
					Extra: map[string]string{
						"Binaries": "test",
					},
				},
				{
					Name:   "foo_1.0.1_windows_386.tar.gz",
					Goos:   "windows",
					Goarch: "386",
					Path:   file,
					Extra: map[string]string{
						"Binaries": "test",
					},
				},
			},
			shouldErr(pipe.ErrSkipPublishEnabled.Error()),
		},
		{
			"is draft",
			args{
				&context.Context{
					Parallelism: runtime.NumCPU(),
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Dist:        ".",
						ProjectName: "run-pipe",
						Release: config.Release{
							Draft: true,
						},
						Scoops: []config.Scoop{
							{
								Bucket: config.Repo{
									Owner: "test",
									Name:  "test",
								},
								Description: "A run pipe test formula",
								Homepage:    "https://github.com/goreleaser",
							},
						},
					},
				},
				&DummyClient{},
			},
			[]artifact.Artifact{
				{
					Name:   "foo_1.0.1_windows_amd64.tar.gz",
					Goos:   "windows",
					Goarch: "amd64",
					Path:   file,
					Extra: map[string]string{
						"Binaries": "test",
					},
				},
				{
					Name:   "foo_1.0.1_windows_386.tar.gz",
					Goos:   "windows",
					Goarch: "386",
					Path:   file,
					Extra: map[string]string{
						"Binaries": "test",
					},
				},
			},
			shouldErr("release is marked as draft"),
		},
		{
			"no archive",
			args{
				&context.Context{
					Parallelism: runtime.NumCPU(),
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Dist:        ".",
						ProjectName: "run-pipe",
						Release: config.Release{
							Draft: true,
						},
						Scoops: []config.Scoop{
							{
								Bucket: config.Repo{
									Owner: "test",
									Name:  "test",
								},
								Description: "A run pipe test formula",
								Homepage:    "https://github.com/goreleaser",
							},
						},
					},
				},
				&DummyClient{},
			},
			[]artifact.Artifact{
				{
					Name:   "foo_1.0.1_windows_amd64",
					Goos:   "windows",
					Goarch: "amd64",
					Extra: map[string]string{
						"Binaries": "test",
					},
					Type: artifact.UploadableBinary,
				},
				{
					Name:   "foo_1.0.1_windows_386",
					Goos:   "windows",
					Goarch: "386",
					Extra: map[string]string{
						"Binaries": "test",
					},
					Type: artifact.UploadableBinary,
				},
			},
			shouldErr(ErrNoWindows.Error()),
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
				Parallelism: runtime.NumCPU(),
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
					Release: config.Release{
						GitHub: config.Repo{
							Owner: "test",
							Name:  "test",
						},
					},
					Scoops: []config.Scoop{
						{
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
		},
		{
			"testdata/test_buildmanifest_url_template.json.golden",
			&context.Context{
				Parallelism: runtime.NumCPU(),
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
					Release: config.Release{
						GitHub: config.Repo{
							Owner: "test",
							Name:  "test",
						},
					},
					Scoops: []config.Scoop{
						{
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
		},
	}

	for _, tt := range tests {
		var ctx = tt.ctx
		Pipe{}.Default(ctx)
		out, err := buildManifest(ctx, []artifact.Artifact{
			{
				Name:   "foo_1.0.1_windows_amd64.tar.gz",
				Goos:   "windows",
				Goarch: "amd64",
				Path:   file,
				Extra: map[string]string{
					"Binaries": "test,foo,barrr",
				},
			},
			{
				Name:   "foo_1.0.1_windows_386.tar.gz",
				Goos:   "windows",
				Goarch: "386",
				Path:   file,
				Extra: map[string]string{
					"Binaries": "test,foo,barrr",
				},
			},
		}, ctx.Config.Scoops[0])

		require.NoError(t, err)

		if *update {
			require.NoError(t, ioutil.WriteFile(tt.filename, out.Bytes(), 0655))
		}
		bts, err := ioutil.ReadFile(tt.filename)
		require.NoError(t, err)
		require.Equal(t, string(bts), out.String())
	}
}

type DummyClient struct {
	CreatedFile bool
	Content     string
}

func (client *DummyClient) CreateRelease(ctx *context.Context, body string) (releaseID int64, err error) {
	return
}

func (client *DummyClient) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo config.Repo, content bytes.Buffer, path, msg string) (err error) {
	client.CreatedFile = true
	bts, _ := ioutil.ReadAll(&content)
	client.Content = string(bts)
	return
}

func (client *DummyClient) Upload(ctx *context.Context, releaseID int64, name string, file *os.File) (err error) {
	return
}
