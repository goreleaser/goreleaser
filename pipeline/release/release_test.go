package release

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/clients"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestPipeDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestRunPipe(t *testing.T) {
	assert := assert.New(t)
	folder, err := ioutil.TempDir("", "gorelasertest")
	assert.NoError(err)
	_, err = os.Create(filepath.Join(folder, "bin.tar.gz"))
	assert.NoError(err)
	_, err = os.Create(filepath.Join(folder, "bin.deb"))
	assert.NoError(err)
	var ctx = &context.Context{
		Git: context.GitInfo{
			CurrentTag: "v1.0.0",
		},
		Config: config.Project{
			Dist: folder,
			Archive: config.Archive{
				Format: "tar.gz",
			},
			Release: config.Release{
				GitHub: config.Repo{
					Owner: "test",
					Name:  "test",
				},
			},
			FPM: config.FPM{
				Formats: []string{
					"deb",
				},
			},
		},
		Archives: map[string]string{
			"darwinamd64": "bin",
		},
	}
	client := &DummyClient{}
	assert.NoError(doRun(ctx, client))
	assert.True(client.CreatedRelease)
	assert.True(client.UploadedFile)
}

type DummyClient struct {
	CreatedRelease bool
	UploadedFile   bool
}

func (client *DummyClient) GetInfo(ctx *context.Context) (info clients.Info, err error) {
	return
}

func (client *DummyClient) CreateRelease(ctx *context.Context) (releaseID int, err error) {
	client.CreatedRelease = true
	return
}

func (client *DummyClient) CreateFile(ctx *context.Context, content bytes.Buffer, path string) (err error) {
	return
}

func (client *DummyClient) Upload(ctx *context.Context, releaseID int, name string, file *os.File) (err error) {
	client.UploadedFile = true
	return
}
