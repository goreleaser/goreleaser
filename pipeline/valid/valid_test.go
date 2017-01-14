package valid

import (
	"testing"

	"github.com/goreleaser/releaser/config"
	"github.com/goreleaser/releaser/context"
	"github.com/stretchr/testify/assert"
)

func runPipe(repo, bin string) error {
	var config = &config.ProjectConfig{
		Repo:       repo,
		BinaryName: bin,
	}
	var ctx = &context.Context{
		Config: config,
	}
	return Pipe{}.Run(ctx)
}

func TestValidadeMissingBinaryName(t *testing.T) {
	assert.Error(t, runPipe("a/b", ""))
}

func TestValidadeMinimalConfig(t *testing.T) {
	assert.NoError(t, runPipe("", "a"))
}
