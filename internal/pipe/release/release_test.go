package release

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestPipeDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestRunPipe(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	assert.NoError(t, err)
	debfile, err := os.Create(filepath.Join(folder, "bin.deb"))
	assert.NoError(t, err)
	var config = config.Project{
		Dist: folder,
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
		},
	}
	var ctx = context.New(config)
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
	})
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debfile.Name(),
	})
	client := &DummyClient{}
	assert.NoError(t, doPublish(ctx, client))
	assert.True(t, client.CreatedRelease)
	assert.True(t, client.UploadedFile)
	assert.Contains(t, client.UploadedFileNames, "bin.deb")
	assert.Contains(t, client.UploadedFileNames, "bin.tar.gz")
}

func TestRunPipeReleaseCreationFailed(t *testing.T) {
	var config = config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
		},
	}
	var ctx = context.New(config)
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	client := &DummyClient{
		FailToCreateRelease: true,
	}
	assert.Error(t, doPublish(ctx, client))
	assert.False(t, client.CreatedRelease)
	assert.False(t, client.UploadedFile)
}

func TestRunPipeWithFileThatDontExist(t *testing.T) {
	var config = config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
		},
	}
	var ctx = context.New(config)
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: "/nope/nope/nope",
	})
	client := &DummyClient{}
	assert.Error(t, doPublish(ctx, client))
	assert.True(t, client.CreatedRelease)
	assert.False(t, client.UploadedFile)
}

func TestRunPipeUploadFailure(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	assert.NoError(t, err)
	var config = config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
		},
	}
	var ctx = context.New(config)
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
	})
	client := &DummyClient{
		FailToUpload: true,
	}
	assert.EqualError(t, doPublish(ctx, client), "failed to upload bin.tar.gz after 10 retries: upload failed")
	assert.True(t, client.CreatedRelease)
	assert.False(t, client.UploadedFile)
}

func TestRunPipeUploadRetry(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	assert.NoError(t, err)
	var config = config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
		},
	}
	var ctx = context.New(config)
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
	})
	client := &DummyClient{
		FailFirstUpload: true,
	}
	assert.NoError(t, doPublish(ctx, client))
	assert.True(t, client.CreatedRelease)
	assert.True(t, client.UploadedFile)
}

func TestPipeDisabled(t *testing.T) {
	var ctx = context.New(config.Project{
		Release: config.Release{
			Disable: true,
		},
	})
	client := &DummyClient{}
	testlib.AssertSkipped(t, doPublish(ctx, client))
	assert.False(t, client.CreatedRelease)
	assert.False(t, client.UploadedFile)
}

func TestDefault(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")

	var ctx = context.New(config.Project{})
	ctx.TokenType = context.TokenTypeGitHub
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Name)
	assert.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Owner)
}

func TestDefaultWithGitlab(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@gitlab.com:gitlabowner/gitlabrepo.git")

	var ctx = context.New(config.Project{})
	ctx.TokenType = context.TokenTypeGitLab
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "gitlabrepo", ctx.Config.Release.GitLab.Name)
	assert.Equal(t, "gitlabowner", ctx.Config.Release.GitLab.Owner)
}

func TestDefaultPreReleaseAuto(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")

	t.Run("auto-release", func(t *testing.T) {
		var ctx = context.New(config.Project{
			Release: config.Release{
				Prerelease: "auto",
			},
		})
		ctx.TokenType = context.TokenTypeGitHub
		ctx.Semver = context.Semver{
			Major: 1,
			Minor: 0,
			Patch: 0,
		}
		assert.NoError(t, Pipe{}.Default(ctx))
		assert.Equal(t, false, ctx.PreRelease)
	})

	t.Run("auto-rc", func(t *testing.T) {
		var ctx = context.New(config.Project{
			Release: config.Release{
				Prerelease: "auto",
			},
		})
		ctx.TokenType = context.TokenTypeGitHub
		ctx.Semver = context.Semver{
			Major:      1,
			Minor:      0,
			Patch:      0,
			Prerelease: "rc1",
		}
		assert.NoError(t, Pipe{}.Default(ctx))
		assert.Equal(t, true, ctx.PreRelease)
	})

	t.Run("auto-rc-github-setup", func(t *testing.T) {
		var ctx = context.New(config.Project{
			Release: config.Release{
				GitHub: config.Repo{
					Name:  "foo",
					Owner: "foo",
				},
				Prerelease: "auto",
			},
		})
		ctx.TokenType = context.TokenTypeGitHub
		ctx.Semver = context.Semver{
			Major:      1,
			Minor:      0,
			Patch:      0,
			Prerelease: "rc1",
		}
		assert.NoError(t, Pipe{}.Default(ctx))
		assert.Equal(t, true, ctx.PreRelease)
	})
}

func TestDefaultPipeDisabled(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")

	var ctx = context.New(config.Project{
		Release: config.Release{
			Disable: true,
		},
	})
	ctx.TokenType = context.TokenTypeGitHub
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Name)
	assert.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Owner)
}

func TestDefaultFilled(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")

	var ctx = &context.Context{
		Config: config.Project{
			Release: config.Release{
				GitHub: config.Repo{
					Name:  "foo",
					Owner: "bar",
				},
			},
		},
	}
	ctx.TokenType = context.TokenTypeGitHub
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "foo", ctx.Config.Release.GitHub.Name)
	assert.Equal(t, "bar", ctx.Config.Release.GitHub.Owner)
}

func TestDefaultNotAGitRepo(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	var ctx = &context.Context{
		Config: config.Project{},
	}
	ctx.TokenType = context.TokenTypeGitHub
	assert.EqualError(t, Pipe{}.Default(ctx), "current folder is not a git repository")
	assert.Empty(t, ctx.Config.Release.GitHub.String())
}

func TestDefaultGitRepoWithoutOrigin(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	var ctx = &context.Context{
		Config: config.Project{},
	}
	ctx.TokenType = context.TokenTypeGitHub
	testlib.GitInit(t)
	assert.EqualError(t, Pipe{}.Default(ctx), "repository doesn't have an `origin` remote")
	assert.Empty(t, ctx.Config.Release.GitHub.String())
}

func TestDefaultNotAGitRepoSnapshot(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	var ctx = &context.Context{
		Config: config.Project{},
	}
	ctx.TokenType = context.TokenTypeGitHub
	ctx.Snapshot = true
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Empty(t, ctx.Config.Release.GitHub.String())
}

func TestDefaultGitRepoWithoutRemote(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	var ctx = &context.Context{
		Config: config.Project{},
	}
	ctx.TokenType = context.TokenTypeGitHub
	assert.Error(t, Pipe{}.Default(ctx))
	assert.Empty(t, ctx.Config.Release.GitHub.String())
}

func TestDefaultMultipleReleasesDefined(t *testing.T) {
	var ctx = context.New(config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "githubName",
				Name:  "githubName",
			},
			GitLab: config.Repo{
				Owner: "gitlabOwner",
				Name:  "gitlabName",
			},
		},
	})
	assert.EqualError(t, Pipe{}.Default(ctx), ErrMultipleReleases.Error())
}

type DummyClient struct {
	FailToCreateRelease bool
	FailToUpload        bool
	CreatedRelease      bool
	UploadedFile        bool
	UploadedFileNames   []string
	FailFirstUpload     bool
	Lock                sync.Mutex
}

func (client *DummyClient) CreateRelease(ctx *context.Context, body string) (releaseID string, err error) {
	if client.FailToCreateRelease {
		return "", errors.New("release failed")
	}
	client.CreatedRelease = true
	return
}

func (client *DummyClient) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo config.Repo, content []byte, path, msg string) (err error) {
	return
}

func (client *DummyClient) Upload(ctx *context.Context, releaseID string, name string, file *os.File) error {
	client.Lock.Lock()
	defer client.Lock.Unlock()
	// ensure file is read to better mimic real behavior
	_, err := ioutil.ReadAll(file)
	if err != nil {
		return errors.Wrapf(err, "unexpected error")
	}
	if client.FailToUpload {
		return errors.New("upload failed")
	}
	if client.FailFirstUpload {
		client.FailFirstUpload = false
		return errors.New("upload failed, should retry")
	}
	client.UploadedFile = true
	client.UploadedFileNames = append(client.UploadedFileNames, name)
	return nil
}
