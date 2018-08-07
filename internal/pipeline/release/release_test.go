package release

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
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
	assert.NoError(t, doRun(ctx, client))
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
	assert.Error(t, doRun(ctx, client))
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
	assert.Error(t, doRun(ctx, client))
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
	assert.Error(t, doRun(ctx, client))
	assert.True(t, client.CreatedRelease)
	assert.False(t, client.UploadedFile)
}

func TestSnapshot(t *testing.T) {
	var ctx = &context.Context{
		SkipPublish: true,
		Parallelism: 1,
	}
	client := &DummyClient{}
	testlib.AssertSkipped(t, doRun(ctx, client))
	assert.False(t, client.CreatedRelease)
	assert.False(t, client.UploadedFile)
}

func TestPipeDisabled(t *testing.T) {
	var ctx = context.New(config.Project{
		Release: config.Release{
			Disable: true,
		},
	})
	client := &DummyClient{}
	testlib.AssertSkipped(t, doRun(ctx, client))
	assert.False(t, client.CreatedRelease)
	assert.False(t, client.UploadedFile)
}

func TestDefault(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")

	var ctx = context.New(config.Project{})
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Name)
	assert.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Owner)
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
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "", ctx.Config.Release.GitHub.Name)
	assert.Equal(t, "", ctx.Config.Release.GitHub.Owner)
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
	assert.EqualError(t, Pipe{}.Default(ctx), "current folder is not a git repository")
	assert.Empty(t, ctx.Config.Release.GitHub.String())
}

func TestDefaultGitRepoWithoutOrigin(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	var ctx = &context.Context{
		Config: config.Project{},
	}
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
	assert.Error(t, Pipe{}.Default(ctx))
	assert.Empty(t, ctx.Config.Release.GitHub.String())
}

type DummyClient struct {
	FailToCreateRelease bool
	FailToUpload        bool
	CreatedRelease      bool
	UploadedFile        bool
	UploadedFileNames   []string
	Lock                sync.Mutex
}

func (client *DummyClient) CreateRelease(ctx *context.Context, body string) (releaseID int64, err error) {
	if client.FailToCreateRelease {
		return 0, errors.New("release failed")
	}
	client.CreatedRelease = true
	return
}

func (client *DummyClient) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo config.Repo, content bytes.Buffer, path, msg string) (err error) {
	return
}

func (client *DummyClient) Upload(ctx *context.Context, releaseID int64, name string, file *os.File) (err error) {
	client.Lock.Lock()
	defer client.Lock.Unlock()
	if client.FailToUpload {
		return errors.New("upload failed")
	}
	client.UploadedFile = true
	client.UploadedFileNames = append(client.UploadedFileNames, name)
	return
}
