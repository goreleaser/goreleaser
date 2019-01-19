package tmpl

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/assert"
)

func TestWithArtifact(t *testing.T) {
	var ctx = context.New(config.Project{
		ProjectName: "proj",
	})
	ctx.Env = map[string]string{
		"FOO": "bar",
	}
	ctx.Version = "1.2.3"
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Semver = context.Semver{
		Major: 1,
		Minor: 2,
		Patch: 3,
	}
	ctx.Git.Commit = "commit"
	ctx.Git.FullCommit = "fullcommit"
	ctx.Git.ShortCommit = "shortcommit"
	for expect, tmpl := range map[string]string{
		"bar":         "{{.Env.FOO}}",
		"Linux":       "{{.Os}}",
		"amd64":       "{{.Arch}}",
		"6":           "{{.Arm}}",
		"1.2.3":       "{{.Version}}",
		"v1.2.3":      "{{.Tag}}",
		"1-2-3":       "{{.Major}}-{{.Minor}}-{{.Patch}}",
		"commit":      "{{.Commit}}",
		"fullcommit":  "{{.FullCommit}}",
		"shortcommit": "{{.ShortCommit}}",
		"binary":      "{{.Binary}}",
		"proj":        "{{.ProjectName}}",
	} {
		tmpl := tmpl
		expect := expect
		t.Run(expect, func(tt *testing.T) {
			tt.Parallel()
			result, err := New(ctx).WithArtifact(
				artifact.Artifact{
					Name:   "not-this-binary",
					Goarch: "amd64",
					Goos:   "linux",
					Goarm:  "6",
					Extra: map[string]interface{}{
						"Binary": "binary",
					},
				},
				map[string]string{"linux": "Linux"},
			).Apply(tmpl)
			assert.NoError(tt, err)
			assert.Equal(tt, expect, result)
		})
	}

	t.Run("artifact without binary name", func(tt *testing.T) {
		tt.Parallel()
		result, err := New(ctx).WithArtifact(
			artifact.Artifact{
				Name:   "another-binary",
				Goarch: "amd64",
				Goos:   "linux",
				Goarm:  "6",
			}, map[string]string{},
		).Apply("{{ .Binary }}")
		assert.NoError(tt, err)
		assert.Equal(tt, ctx.Config.ProjectName, result)
	})

	t.Run("template using artifact fields with no artifact", func(tt *testing.T) {
		tt.Parallel()
		result, err := New(ctx).Apply("{{ .Os }}")
		assert.EqualError(tt, err, `template: tmpl:1:3: executing "tmpl" at <.Os>: map has no entry for key "Os"`)
		assert.Empty(tt, result)
	})
}

func TestEnv(t *testing.T) {
	testCases := []struct {
		desc string
		in   string
		out  string
	}{
		{
			desc: "with env",
			in:   "{{ .Env.FOO }}",
			out:  "BAR",
		},
		{
			desc: "with env",
			in:   "{{ .Env.BAR }}",
			out:  "",
		},
	}
	var ctx = context.New(config.Project{})
	ctx.Env = map[string]string{
		"FOO": "BAR",
	}
	ctx.Git.CurrentTag = "v1.2.3"
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			out, _ := New(ctx).Apply(tC.in)
			assert.Equal(t, tC.out, out)
		})
	}
}

func TestFuncMap(t *testing.T) {
	var ctx = context.New(config.Project{
		ProjectName: "proj",
	})
	ctx.Git.CurrentTag = "v1.2.4"
	for _, tc := range []struct {
		Template string
		Name     string
	}{
		{
			Template: `{{ time "2006-01-02" }}`,
			Name:     "YYYY-MM-DD",
		},
		{
			Template: `{{ time "01/02/2006" }}`,
			Name:     "MM/DD/YYYY",
		},
		{
			Template: `{{ time "01/02/2006" }}`,
			Name:     "MM/DD/YYYY",
		},
	} {
		out, err := New(ctx).Apply(tc.Template)
		assert.NoError(t, err)
		assert.NotEmpty(t, out)
	}
}

func TestInvalidTemplate(t *testing.T) {
	ctx := context.New(config.Project{})
	ctx.Git.CurrentTag = "v1.1.1"
	_, err := New(ctx).Apply("{{{.Foo}")
	assert.EqualError(t, err, "template: tmpl:1: unexpected \"{\" in command")
}

func TestEnvNotFound(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.Git.CurrentTag = "v1.2.4"
	result, err := New(ctx).Apply("{{.Env.FOO}}")
	assert.Empty(t, result)
	assert.EqualError(t, err, `template: tmpl:1:6: executing "tmpl" at <.Env.FOO>: map has no entry for key "FOO"`)
}
