package valid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidadeMissingBinaryName(t *testing.T) {
	assert := assert.New(t)

	config := ProjectConfig{Repo: "asd/asd"}
	assert.Error(config.validate())
}

func TestValidadeMissingRepo(t *testing.T) {
	assert := assert.New(t)

	config := ProjectConfig{BinaryName: "asd"}
	assert.Error(config.validate())
}

func TestValidadeMinimalConfig(t *testing.T) {
	assert := assert.New(t)

	config := ProjectConfig{BinaryName: "asd", Repo: "asd/asd"}
	assert.NoError(config.validate())
}
