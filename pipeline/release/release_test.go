package release

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
)

func TestPipeDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestRunPipe(t *testing.T) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(err)
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	assert.NoError(err)
	debfile, err := os.Create(filepath.Join(folder, "bin.deb"))
	assert.NoError(err)
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
	ctx.Publish = true
	ctx.AddArtifact(tarfile.Name())
	ctx.AddArtifact(debfile.Name())
	client := &DummyClient{}
	assert.NoError(doRun(ctx, client))
	assert.True(client.CreatedRelease)
	assert.True(client.UploadedFile)
	assert.Contains(client.UploadedFileNames, "bin.deb")
	assert.Contains(client.UploadedFileNames, "bin.tar.gz")
}

func TestRunPipeReleaseCreationFailed(t *testing.T) {
	var assert = assert.New(t)
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
	ctx.Publish = true
	client := &DummyClient{
		FailToCreateRelease: true,
	}
	assert.Error(doRun(ctx, client))
	assert.False(client.CreatedRelease)
	assert.False(client.UploadedFile)
}

func TestRunPipeWithFileThatDontExist(t *testing.T) {
	var assert = assert.New(t)
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
	ctx.Publish = true
	ctx.AddArtifact("this-file-wont-exist-hopefully")
	client := &DummyClient{}
	assert.Error(doRun(ctx, client))
	assert.True(client.CreatedRelease)
	assert.False(client.UploadedFile)
}

func TestRunPipeUploadFailure(t *testing.T) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(err)
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	assert.NoError(err)
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
	ctx.Publish = true
	ctx.AddArtifact(tarfile.Name())
	client := &DummyClient{
		FailToUpload: true,
	}
	assert.Error(doRun(ctx, client))
	assert.True(client.CreatedRelease)
	assert.False(client.UploadedFile)
}

func TestSkipPublish(t *testing.T) {
	var assert = assert.New(t)
	var ctx = &context.Context{
		Publish:     false,
		Parallelism: 1,
	}
	client := &DummyClient{}
	testlib.AssertSkipped(t, doRun(ctx, client))
	assert.False(client.CreatedRelease)
	assert.False(client.UploadedFile)
}

type DummyClient struct {
	FailToCreateRelease bool
	FailToUpload        bool
	CreatedRelease      bool
	UploadedFile        bool
	UploadedFileNames   []string
}

func (client *DummyClient) CreateRelease(ctx *context.Context, body string) (releaseID int, err error) {
	if client.FailToCreateRelease {
		return 0, errors.New("release failed")
	}
	client.CreatedRelease = true
	return
}

func (client *DummyClient) CreateFile(ctx *context.Context, content bytes.Buffer, path string) (err error) {
	return
}

func (client *DummyClient) Upload(ctx *context.Context, releaseID int, name string, file *os.File) (err error) {
	if client.FailToUpload {
		return errors.New("upload failed")
	}
	client.UploadedFile = true
	client.UploadedFileNames = append(client.UploadedFileNames, name)
	return
}
