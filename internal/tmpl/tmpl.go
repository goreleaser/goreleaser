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
	kProjectName = "ProjectName"
	kVersion     = "Version"
	kTag         = "Tag"
	kCommit      = "Commit"
	kMajor       = "Major"
	kMinor       = "Minor"
	kPatch       = "Patch"
	kEnv         = "Env"

	// artifact-only keys
	kOs     = "Os"
	kArch   = "Arch"
	kArm    = "Arm"
	kBinary = "Binary"
)

// New Template
func New(ctx *context.Context) *Template {
	return &Template{
		fields: fields{
			kProjectName: ctx.Config.ProjectName,
			kVersion:     ctx.Version,
			kTag:         ctx.Git.CurrentTag,
			kCommit:      ctx.Git.Commit,
			kEnv:         ctx.Env,
		},
	}
}

// WithArtifacts populate fields from the artifact and replacements
func (t *Template) WithArtifact(a artifact.Artifact, replacements map[string]string) *Template {
	var binary = a.Extra[kBinary]
	if binary == "" {
		binary = t.fields[kProjectName].(string)
	}
	t.fields[kOs] = replace(replacements, a.Goos)
	t.fields[kArch] = replace(replacements, a.Goarch)
	t.fields[kArm] = replace(replacements, a.Goarm)
	t.fields[kBinary] = binary
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

	sv, err := semver.NewVersion(t.fields[kTag].(string))
	if err != nil {
		return "", errors.Wrap(err, "tmpl")
	}
	t.fields[kMajor] = sv.Major()
	t.fields[kMinor] = sv.Minor()
	t.fields[kPatch] = sv.Patch()

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
