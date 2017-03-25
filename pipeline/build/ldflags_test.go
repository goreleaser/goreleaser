package build

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestLdFlagsFullTemplate(t *testing.T) {
	assert := assert.New(t)
	var config = config.Project{
		Build: config.Build{
			Ldflags: "-s -w -X main.version={{.Version}} -X main.date={{.Date}} -X main.commit={{.Commit}}",
		},
	}
	var ctx = &context.Context{
		Git: context.GitInfo{
			CurrentTag: "v1.2.3",
			Commit:     "123",
		},
		Config: config,
	}
	flags, err := ldflags(ctx)
	assert.NoError(err)
	assert.Contains(flags, "-s -w")
	assert.Contains(flags, "-X main.version=v1.2.3")
	assert.Contains(flags, "-X main.commit=123")
	// TODO assert main.date
}

func TestInvalidTemplate(t *testing.T) {
	assert := assert.New(t)
	var config = config.Project{
		Build: config.Build{
			Ldflags: "{invalid{.Template}}}{{}}}",
		},
	}
	var ctx = &context.Context{
		Config: config,
	}
	flags, err := ldflags(ctx)
	assert.Error(err)
	assert.Equal(flags, "")
}
