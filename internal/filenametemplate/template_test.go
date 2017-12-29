package filenametemplate

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/stretchr/testify/assert"
)

func TestTemplate(t *testing.T) {
	var ctx = context.New(config.Project{
		ProjectName: "proj",
	})
	ctx.Env = map[string]string{
		"FOO": "bar",
	}
	ctx.Version = "1.0.0"
	ctx.Git.CurrentTag = "v1.0.0"
	var artifact = artifact.Artifact{
		Name:   "not-this-binary",
		Goarch: "amd64",
		Goos:   "linux",
		Goarm:  "6",
		Extra: map[string]string{
			"Binary": "binary",
		},
	}
	var fields = NewFields(ctx, map[string]string{"linux": "Linux"}, artifact)
	for expect, tmpl := range map[string]string{
		"bar":    "{{.Env.FOO}}",
		"Linux":  "{{.Os}}",
		"amd64":  "{{.Arch}}",
		"6":      "{{.Arm}}",
		"1.0.0":  "{{.Version}}",
		"v1.0.0": "{{.Tag}}",
		"binary": "{{.Binary}}",
		"proj":   "{{.ProjectName}}",
	} {
		tmpl := tmpl
		expect := expect
		t.Run(expect, func(tt *testing.T) {
			tt.Parallel()
			result, err := Apply(tmpl, fields)
			assert.NoError(tt, err)
			assert.Equal(tt, expect, result)
		})
	}
}

func TestNewFields(t *testing.T) {
	var ctx = context.New(config.Project{
		ProjectName: "proj",
	})
	ctx.Version = "1.0.0"
	ctx.Git.CurrentTag = "v1.0.0"
	var artifact = artifact.Artifact{
		Name:   "not-this-binary",
		Goarch: "amd64",
		Goos:   "linux",
		Goarm:  "6",
		Extra: map[string]string{
			"Binary": "binary",
		},
	}
	var fields = NewFields(ctx, map[string]string{}, artifact, artifact)
	assert.Equal(t, "proj", fields.Binary)
}

func TestInvalidTemplate(t *testing.T) {
	var ctx = context.New(config.Project{})
	var fields = NewFields(ctx, map[string]string{}, artifact.Artifact{})
	result, err := Apply("{{.Foo}", fields)
	assert.Empty(t, result)
	assert.EqualError(t, err, `template: {{.Foo}:1: unexpected "}" in operand`)
}
