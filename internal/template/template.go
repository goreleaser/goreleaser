// Package template contains the code used to template names of goreleaser's
// packages and archives.
package template

import (
	"bytes"
	gotemplate "text/template"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
)

// Fields contains all accepted fields in the template string
type Fields struct {
	Version     string
	Tag         string
	ProjectName string
	Env         map[string]string
	Os          string
	Arch        string
	Arm         string
	Binary      string
}

// NewFields returns a Fields instances filled with the data provided
func NewFields(ctx *context.Context, a artifact.Artifact, replacements map[string]string) Fields {
	return Fields{
		Env:         ctx.Env,
		Version:     ctx.Version,
		Tag:         ctx.Git.CurrentTag,
		ProjectName: ctx.Config.ProjectName,
		Binary:      a.Extra["Binary"],
		Os:          replace(replacements, a.Goos),
		Arch:        replace(replacements, a.Goarch),
		Arm:         replace(replacements, a.Goarm),
	}
}

// Apply applies the given fields to the given template and returns the
// evaluation and any error that might occur.
func Apply(tmpl string, fields Fields) (string, error) {
	t, err := gotemplate.New(tmpl).Parse(tmpl)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	err = t.Execute(&out, fields)
	return out.String(), err
}

func replace(replacements map[string]string, original string) string {
	result := replacements[original]
	if result == "" {
		return original
	}
	return result
}
