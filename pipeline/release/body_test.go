package release

import (
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestDescribeBody(t *testing.T) {
	var assert = assert.New(t)
	var changelog = "\nfeature1: description\nfeature2: other description"
	var ctx = &context.Context{
		ReleaseNotes: changelog,
	}
	out, err := describeBody(ctx)
	assert.NoError(err)
	assert.Contains(out.String(), changelog)
	assert.Contains(out.String(), "Automated with [GoReleaser]")
	assert.Contains(out.String(), "Built with go version go1.")
}

func TestDontEscapeHTML(t *testing.T) {
	var assert = assert.New(t)
	var changelog = "<h1>test</h1>"
	var ctx = &context.Context{
		ReleaseNotes: changelog,
	}
	out, err := describeBody(ctx)
	assert.NoError(err)
	assert.Contains(out.String(), changelog)
}

func TestGoVersionFails(t *testing.T) {
	var assert = assert.New(t)
	var path = os.Getenv("PATH")
	defer func() {
		assert.NoError(os.Setenv("PATH", path))
	}()
	assert.NoError(os.Setenv("PATH", ""))
	var ctx = &context.Context{
		ReleaseNotes: "changelog",
	}
	_, err := describeBody(ctx)
	assert.Error(err)
}
