package release

import (
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestBody(t *testing.T) {
	var assert = assert.New(t)
	var changelog = "\nfeature1: description\nfeature2: other description"
	var ctx = &context.Context{
		ReleaseNotes: changelog,
	}
	out, err := buildBody(ctx)
	assert.NoError(err)
	assert.Contains(out.String(), changelog)
	assert.Contains(out.String(), "Automated with @goreleaser")
	assert.Contains(out.String(), "Built with go version go1.8")
}

func TestGoVersionFails(t *testing.T) {
	var assert = assert.New(t)
	var path = os.Getenv("PATH")
	defer func() {
		assert.NoError(os.Setenv("PATH", path))
	}()
	os.Setenv("PATH", "")
	var ctx = &context.Context{
		ReleaseNotes: "changelog",
	}
	_, err := buildBody(ctx)
	assert.Error(err)
}
