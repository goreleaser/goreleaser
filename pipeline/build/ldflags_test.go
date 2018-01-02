package build

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestLdFlagsFullTemplate(t *testing.T) {
	var config = config.Project{
		Builds: []config.Build{
			{
				Ldflags: `-s -w -X main.version={{.Version}} -X main.tag={{.Tag}} -X main.date={{.Date}} -X main.commit={{.Commit}} -X "main.foo={{.Env.FOO}}"`,
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
		Env:     map[string]string{"FOO": "123"},
	}
	flags, err := ldflags(ctx, ctx.Config.Builds[0])
	assert.NoError(t, err)
	assert.Contains(t, flags, "-s -w")
	assert.Contains(t, flags, "-X main.version=1.2.3")
	assert.Contains(t, flags, "-X main.tag=v1.2.3")
	assert.Contains(t, flags, "-X main.commit=123")
	assert.Contains(t, flags, "-X main.date=")
	assert.Contains(t, flags, `-X "main.foo=123"`)
}

func TestInvalidTemplate(t *testing.T) {
	for template, eerr := range map[string]string{
		"{{ .Nope }":    `template: ldflags:1: unexpected "}" in operand`,
		"{{.Env.NOPE}}": `template: ldflags:1:6: executing "ldflags" at <.Env.NOPE>: map has no entry for key "NOPE"`,
	} {
		t.Run(template, func(tt *testing.T) {
			var config = config.Project{
				Builds: []config.Build{
					{Ldflags: template},
				},
			}
			var ctx = &context.Context{
				Config: config,
			}
			flags, err := ldflags(ctx, ctx.Config.Builds[0])
			assert.EqualError(tt, err, eerr)
			assert.Empty(tt, flags)
		})
	}
}
