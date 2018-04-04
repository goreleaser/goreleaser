package before

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestRunPipe(t *testing.T) {
	for _, tc := range [][]string{
		nil,
		{},
		{"go version"},
		{"go version", "go list"},
	} {
		ctx := context.New(
			config.Project{
				Before: config.Before{
					Hooks: tc,
				},
			},
		)
		assert.NoError(t, Pipe{}.Run(ctx))
	}
}

func TestRunPipeFail(t *testing.T) {
	for _, tc := range [][]string{
		{"go tool foobar"},
	} {
		ctx := context.New(
			config.Project{
				Before: config.Before{
					Hooks: tc,
				},
			},
		)
		assert.Error(t, Pipe{}.Run(ctx))
	}
}
