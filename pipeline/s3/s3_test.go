package s3

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestNoS3(t *testing.T) {
	assert.NoError(t, Pipe{}.Run(context.New(config.Project{})))
}

func TestDefaultsNoS3(t *testing.T) {
	var assert = assert.New(t)
	var ctx = context.New(config.Project{
		S3: []config.S3{
			config.S3{},
		},
	})
	assert.NoError(Pipe{}.Default(ctx))
	assert.Equal([]config.S3{config.S3{}}, ctx.Config.S3)
}

func TestDefaults(t *testing.T) {
	var assert = assert.New(t)
	var ctx = context.New(config.Project{
		S3: []config.S3{
			{
				Bucket: "foo",
			},
		},
	})
	assert.NoError(Pipe{}.Default(ctx))
	assert.Equal([]config.S3{config.S3{
		Bucket: "foo",
		Region: "us-east-1",
		Folder: "{{ .ProjectName }}/{{ .Tag }}",
	}}, ctx.Config.S3)
}
