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
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: d,
			Type: artifact.DockerImage,
		})
	}
	out, err := describeBody(ctx)
	assert.NoError(t, err)

	var golden = "testdata/release1.golden"
	if *update {
		_ = ioutil.WriteFile(golden, out.Bytes(), 0755)
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
		_ = ioutil.WriteFile(golden, out.Bytes(), 0655)
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

func TestAddHeaderToDescription(t *testing.T) {
	assert := assert.New(t)

	var changelog = "feature1: description\nfeature2: other description"
	var ctx = context.New(config.Project{
		Release: config.Release{
			HeaderTemplate: "Header template",
		},
	})

	ctx.ReleaseNotes = changelog
	for _, d := range []string{
		"goreleaser/goreleaser:0.40.0",
		"goreleaser/goreleaser:latest",
		"goreleaser/godownloader:v0.1.0",
	} {
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: d,
			Type: artifact.DockerImage,
		})
	}
	out, err := describeBody(ctx)
	assert.NoError(err)

	var golden = "testdata/release3.golden"
	if *update {
		_ = ioutil.WriteFile(golden, out.Bytes(), 0755)
	}
	bts, err := ioutil.ReadFile(golden)
	assert.NoError(err)
	assert.Equal(string(bts), out.String())
}

func TestAddFooterToDescription(t *testing.T) {
	assert := assert.New(t)

	var changelog = "feature1: description\nfeature2: other description"
	var ctx = context.New(config.Project{
		Release: config.Release{
			FooterTemplate: "Footer template",
		},
	})

	ctx.ReleaseNotes = changelog
	for _, d := range []string{
		"goreleaser/goreleaser:0.40.0",
		"goreleaser/goreleaser:latest",
		"goreleaser/godownloader:v0.1.0",
	} {
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: d,
			Type: artifact.DockerImage,
		})
	}
	out, err := describeBody(ctx)
	assert.NoError(err)

	var golden = "testdata/release4.golden"
	if *update {
		_ = ioutil.WriteFile(golden, out.Bytes(), 0755)
	}
	bts, err := ioutil.ReadFile(golden)
	assert.NoError(err)
	assert.Equal(string(bts), out.String())
}

func TestAddHeaderAndFooterToDescription(t *testing.T) {
	assert := assert.New(t)

	var changelog = "feature1: description\nfeature2: other description"
	var ctx = context.New(config.Project{
		Release: config.Release{
			HeaderTemplate: "Header template",
			FooterTemplate: "Footer template",
		},
	})

	ctx.ReleaseNotes = changelog
	for _, d := range []string{
		"goreleaser/goreleaser:0.40.0",
		"goreleaser/goreleaser:latest",
		"goreleaser/godownloader:v0.1.0",
	} {
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: d,
			Type: artifact.DockerImage,
		})
	}
	out, err := describeBody(ctx)
	assert.NoError(err)

	var golden = "testdata/release5.golden"
	if *update {
		_ = ioutil.WriteFile(golden, out.Bytes(), 0755)
	}
	bts, err := ioutil.ReadFile(golden)
	assert.NoError(err)
	assert.Equal(string(bts), out.String())
}

func TestDescribeBodyNoDockerImagesNoBrewsWithHeader(t *testing.T) {
	var changelog = "feature1: description\nfeature2: other description"
	var ctx = &context.Context{
		ReleaseNotes: changelog,
		Config: config.Project{
			Release: config.Release{
				HeaderTemplate: "Header template",
			}},
	}
	out, err := describeBody(ctx)
	assert.NoError(t, err)

	var golden = "testdata/release6.golden"
	if *update {
		_ = ioutil.WriteFile(golden, out.Bytes(), 0655)
	}
	bts, err := ioutil.ReadFile(golden)
	assert.NoError(t, err)

	assert.Equal(t, string(bts), out.String())
}

func TestDescribeBodyNoDockerImagesNoBrewsWithFooter(t *testing.T) {
	var changelog = "feature1: description\nfeature2: other description"
	var ctx = &context.Context{
		ReleaseNotes: changelog,
		Config: config.Project{
			Release: config.Release{
				FooterTemplate: "Footer template",
			}},
	}
	out, err := describeBody(ctx)
	assert.NoError(t, err)

	var golden = "testdata/release7.golden"
	if *update {
		_ = ioutil.WriteFile(golden, out.Bytes(), 0655)
	}
	bts, err := ioutil.ReadFile(golden)
	assert.NoError(t, err)

	assert.Equal(t, string(bts), out.String())
}

func TestDescribeBodyNoDockerImagesNoBrewsWithHeaderAndFooter(t *testing.T) {
	var changelog = "feature1: description\nfeature2: other description"
	var ctx = &context.Context{
		ReleaseNotes: changelog,
		Config: config.Project{
			Release: config.Release{
				HeaderTemplate: "Header template",
				FooterTemplate: "Footer template",
			}},
	}
	out, err := describeBody(ctx)
	assert.NoError(t, err)

	var golden = "testdata/release8.golden"
	if *update {
		_ = ioutil.WriteFile(golden, out.Bytes(), 0655)
	}
	bts, err := ioutil.ReadFile(golden)
	assert.NoError(t, err)

	assert.Equal(t, string(bts), out.String())
}
