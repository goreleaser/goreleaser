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

type fields map[string]interface{}

const (
	// general keys
	projectName = "ProjectName"
	version     = "Version"
	tag         = "Tag"
	commit      = "Commit"
	major       = "Major"
	minor       = "Minor"
	patch       = "Patch"
	env         = "Env"
	date        = "Date"
	timestamp   = "Timestamp"

	// artifact-only keys
	os     = "Os"
	arch   = "Arch"
	arm    = "Arm"
	binary = "Binary"
)

// New Template
func New(ctx *context.Context) *Template {
	return &Template{
		fields: fields{
			projectName: ctx.Config.ProjectName,
			version:     ctx.Version,
			tag:         ctx.Git.CurrentTag,
			commit:      ctx.Git.Commit,
			env:         ctx.Env,
			date:        time.Now().UTC().Format(time.RFC3339),
			timestamp:   time.Now().UTC().Unix(),
		},
	}
}

// WithArtifacts populate fields from the artifact and replacements
func (t *Template) WithArtifact(a artifact.Artifact, replacements map[string]string) *Template {
	var bin = a.Extra[binary]
	if bin == "" {
		bin = t.fields[projectName].(string)
	}
	t.fields[os] = replace(replacements, a.Goos)
	t.fields[arch] = replace(replacements, a.Goarch)
	t.fields[arm] = replace(replacements, a.Goarm)
	t.fields[binary] = bin
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

	sv, err := semver.NewVersion(t.fields[tag].(string))
	if err != nil {
		return "", errors.Wrap(err, "tmpl")
	}
	t.fields[major] = sv.Major()
	t.fields[minor] = sv.Minor()
	t.fields[patch] = sv.Patch()

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
