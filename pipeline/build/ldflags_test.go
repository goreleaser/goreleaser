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
		Builds: []config.Build{
			{
				Ldflags: "-s -w -X main.version={{.Version}} -X main.tag={{.Tag}} -X main.date={{.Date}} -X main.commit={{.Commit}}",
			},
		},
	}
	var ctx = &context.Context{
		Git: context.GitInfo{
			CurrentTag: "v1.2.3",
			Commit:     "123",
		},
		Version: "1.2.3",
		Config:  config,
	}
	flags, err := ldflags(ctx, ctx.Config.Builds[0])
	assert.NoError(err)
	assert.Contains(flags, "-s -w")
	assert.Contains(flags, "-X main.version=1.2.3")
	assert.Contains(flags, "-X main.tag=v1.2.3")
	assert.Contains(flags, "-X main.commit=123")
	assert.Contains(flags, "-X main.date=")
}

func TestInvalidTemplate(t *testing.T) {
	assert := assert.New(t)
	var config = config.Project{
		Builds: []config.Build{
			{Ldflags: "{invalid{.Template}}}{{}}}"},
		},
	}
	var ctx = &context.Context{
		Config: config,
	}
	flags, err := ldflags(ctx, ctx.Config.Builds[0])
	assert.Error(err)
	assert.Equal(flags, "")
}
