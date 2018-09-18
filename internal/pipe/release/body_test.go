package release

import (
	"flag"
	"io/ioutil"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/assert"
)

var update = flag.Bool("update", false, "update .golden files")

func TestDescribeBody(t *testing.T) {
	var changelog = "feature1: description\nfeature2: other description"
	var ctx = context.New(config.Project{})
	ctx.ReleaseNotes = changelog
	for _, d := range []string{
		"goreleaser/goreleaser:0.40.0",
		"goreleaser/goreleaser:latest",
		"goreleaser/godownloader:v0.1.0",
	} {
		ctx.Artifacts.Add(artifact.Artifact{
			Name: d,
			Type: artifact.DockerImage,
		})
	}
	out, err := describeBody(ctx)
	assert.NoError(t, err)

	var golden = "testdata/release1.golden"
	if *update {
		ioutil.WriteFile(golden, out.Bytes(), 0755)
	}
	bts, err := ioutil.ReadFile(golden)
	assert.NoError(t, err)
	assert.Equal(t, string(bts), out.String())
}

func TestDescribeBodyNoDockerImagesNoBrews(t *testing.T) {
	var changelog = "feature1: description\nfeature2: other description"
	var ctx = &context.Context{
		ReleaseNotes: changelog,
	}
	out, err := describeBody(ctx)
	assert.NoError(t, err)

	var golden = "testdata/release2.golden"
	if *update {
		ioutil.WriteFile(golden, out.Bytes(), 0655)
	}
	bts, err := ioutil.ReadFile(golden)
	assert.NoError(t, err)

	assert.Equal(t, string(bts), out.String())
}

func TestDontEscapeHTML(t *testing.T) {
	var changelog = "<h1>test</h1>"
	var ctx = context.New(config.Project{})
	ctx.ReleaseNotes = changelog

	out, err := describeBody(ctx)
	assert.NoError(t, err)
	assert.Contains(t, out.String(), changelog)
}
