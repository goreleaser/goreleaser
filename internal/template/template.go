package template

import (
	"bytes"
	gotemplate "text/template"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
)

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

func NewFields(ctx *context.Context, a artifact.Artifact, replacements map[string]string) Fields {
	return Fields{
		Env:         ctx.Env,
		Version:     ctx.Version,
		Tag:         ctx.Git.CurrentTag,
		ProjectName: ctx.Config.ProjectName,
		Binary:      a.Name,
		Os:          replace(replacements, a.Goos),
		Arch:        replace(replacements, a.Goarch),
		Arm:         replace(replacements, a.Goarm),
	}
}

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
