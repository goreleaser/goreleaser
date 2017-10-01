package release

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestDescribeBody(t *testing.T) {
	var changelog = "\nfeature1: description\nfeature2: other description"
	var ctx = &context.Context{
		ReleaseNotes: changelog,
		Dockers: []string{
			"goreleaser/goreleaser:0.40.0",
			"goreleaser/godownloader:0.1.0",
		},
	}
	out, err := describeBodyVersion(ctx, "go version go1.9 darwin/amd64")
	assert.NoError(t, err)

	bts, err := ioutil.ReadFile("testdata/release1.txt")
	assert.NoError(t, err)
	// ioutil.WriteFile("testdata/release1.txt", out.Bytes(), 0755)

	assert.Equal(t, string(bts), out.String())
}

func TestDescribeBodyNoDockerImages(t *testing.T) {
	var changelog = "\nfeature1: description\nfeature2: other description"
	var ctx = &context.Context{
		ReleaseNotes: changelog,
	}
	out, err := describeBodyVersion(ctx, "go version go1.9 darwin/amd64")
	assert.NoError(t, err)

	bts, err := ioutil.ReadFile("testdata/release2.txt")
	assert.NoError(t, err)
	// ioutil.WriteFile("testdata/release2.txt", out.Bytes(), 0755)

	assert.Equal(t, string(bts), out.String())
}

func TestDontEscapeHTML(t *testing.T) {
	var changelog = "<h1>test</h1>"
	var ctx = &context.Context{
		ReleaseNotes: changelog,
	}
	out, err := describeBody(ctx)
	assert.NoError(t, err)
	assert.Contains(t, out.String(), changelog)
}

func TestGoVersionFails(t *testing.T) {
	var path = os.Getenv("PATH")
	defer func() {
		assert.NoError(t, os.Setenv("PATH", path))
	}()
	assert.NoError(t, os.Setenv("PATH", ""))
	var ctx = &context.Context{
		ReleaseNotes: "changelog",
	}
	_, err := describeBody(ctx)
	assert.Error(t, err)
}
