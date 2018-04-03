package source

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestRunPipe(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	defer os.RemoveAll(folder)

	dist := filepath.Join(folder, "dist")
	ctx := context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Builds: []config.Build{
			{
				Binary: "mybin",
			},
		},
		Source: config.Source{
			NameTemplate: "{{.Binary}}-{{.Version}}",
		},
	})
	ctx.Version = "testversion"
	ctx.Git.Commit = "FEFEFEFE"
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Equal(t, "mybin-testversion.tar.gz", ctx.Config.Brew.SourceTarball)

	arts := ctx.Artifacts.Filter(artifact.ByType(artifact.Source))
	assert.Equal(t, 1, len(arts.List()))

	assert.NoError(t, ioutil.WriteFile(filepath.Join(folder, "testfile"), []byte("foobar"), 0600))
	assert.NoError(t, create(ctx, folder))
}
