// Package tmpl provides templating utilities for goreleser
package tmpl

import (
	"bytes"
	"text/template"
	"time"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/masterminds/semver"
	"github.com/pkg/errors"
)

// Template holds data that can be applied to a template string
type Template struct {
	fields fields
}

type fields struct {
	ProjectName string
	Version     string
	Tag         string
	Commit      string
	Major       int64
	Minor       int64
	Patch       int64
	Env         map[string]string

	// artifact-only fields
	Os     string
	Arch   string
	Arm    string
	Binary string
}

// New Template
func New(ctx *context.Context) *Template {
	return &Template{
		fields: fields{
			ProjectName: ctx.Config.ProjectName,
			Version:     ctx.Version,
			Tag:         ctx.Git.CurrentTag,
			Commit:      ctx.Git.Commit,
			Env:         ctx.Env,
		},
	}
}

// WithArtifacts populate fields from the artifact and replacements
func (t *Template) WithArtifact(a artifact.Artifact, replacements map[string]string) *Template {
	var binary = a.Extra["Binary"]
	if binary == "" {
		binary = t.fields.ProjectName
	}
	t.fields.Os = replace(replacements, a.Goos)
	t.fields.Arch = replace(replacements, a.Goarch)
	t.fields.Arm = replace(replacements, a.Goarm)
	t.fields.Binary = binary
	return t
}

// Apply applies the given string against the fields stored in the template.
func (t *Template) Apply(s string) (string, error) {
	var out bytes.Buffer
	tmpl, err := template.New("tmpl").
		Option("missingkey=error").
		Funcs(template.FuncMap{
			"time": func(s string) string {
				return time.Now().UTC().Format(s)
			},
		}).
		Parse(s)
	if err != nil {
		return "", err
	}

	sv, err := semver.NewVersion(t.fields.Tag)
	if err != nil {
		return "", errors.Wrap(err, "tmpl")
	}
	t.fields.Major = sv.Major()
	t.fields.Minor = sv.Minor()
	t.fields.Patch = sv.Patch()

	err = tmpl.Execute(&out, t.fields)
	return out.String(), err
}

func replace(replacements map[string]string, original string) string {
	result := replacements[original]
	if result == "" {
		return original
	}
	return result
}
