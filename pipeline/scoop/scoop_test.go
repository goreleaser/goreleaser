package scoop

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/stretchr/testify/assert"
)

func Test_buildManifest(t *testing.T) {
	type args struct {
		ctx       *context.Context
		client    client.Client
		artifacts []artifact.Artifact
	}
	tests := []struct {
		name       string
		args       args
		wantResult Manifest
		wantErr    bool
	}{
		{
			"simple",
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
				[]artifact.Artifact{
					{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64"},
					{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386"},
				},
			},
			Manifest{
				Version: "1.0.1",
				Architecture: map[string]Resource{
					"64bit": {
						URL: "https://github.com/test/test/releases/download/1.0.1/foo_1.0.1_windows_amd64.tar.gz",
						Bin: "test.exe",
					},
					"32bit": {
						URL: "https://github.com/test/test/releases/download/1.0.1/foo_1.0.1_windows_386.tar.gz",
						Bin: "test.exe",
					},
				},
				Homepage:    "https://github.com/goreleaser",
				Description: "A run pipe test formula",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := buildManifest(tt.args.ctx, tt.args.client, tt.args.artifacts)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			var gotResultAsObject Manifest
			err = json.Unmarshal(gotResult.Bytes(), &gotResultAsObject)
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, tt.wantResult, gotResultAsObject)
		})
	}
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
