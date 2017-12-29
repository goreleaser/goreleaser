// Package filenametemplate contains the code used to template names of
// goreleaser's  packages and archives.
package filenametemplate

import (
	"bytes"
	"text/template"

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
func NewFields(ctx *context.Context, replacements map[string]string, artifacts ...artifact.Artifact) Fields {
	// This will fail if artifacts is empty - should never be though...
	var binary = artifacts[0].Extra["Binary"]
	if len(artifacts) > 1 {
		binary = ctx.Config.ProjectName
	}
	return Fields{
		Env:         ctx.Env,
		Version:     ctx.Version,
		Tag:         ctx.Git.CurrentTag,
		ProjectName: ctx.Config.ProjectName,
		Os:          replace(replacements, artifacts[0].Goos),
		Arch:        replace(replacements, artifacts[0].Goarch),
		Arm:         replace(replacements, artifacts[0].Goarm),
		Binary:      binary,
	}
}

// Apply applies the given fields to the given template and returns the
// evaluation and any error that might occur.
func Apply(tmpl string, fields Fields) (string, error) {
	t, err := template.New(tmpl).Parse(tmpl)
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
