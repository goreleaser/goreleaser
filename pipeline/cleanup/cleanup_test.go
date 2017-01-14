package cleanup

import (
	"testing"

	"github.com/goreleaser/releaser/config"
	"github.com/stretchr/testify/assert"
)

func TestDeletesFolder(t *testing.T) {
	assert.NoError(t, Pipe{}.Run(config.ProjectConfig{}))
}

func TestName(t *testing.T) {
	assert.Equal(t, "Cleanup", Pipe{}.Name())
}
