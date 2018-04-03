package source

import (
	"bytes"
	"text/template"

	"github.com/goreleaser/goreleaser/context"
)

type nameData struct {
	Version string
	Tag     string
	Binary  string
}

func nameFor(ctx *context.Context) (string, error) {
	var data = nameData{
		Version: ctx.Version,
		Tag:     ctx.Git.CurrentTag,
		Binary:  ctx.Config.Builds[0].Binary,
	}

	var out bytes.Buffer
	t, err := template.New(data.Binary).Parse(ctx.Config.Source.NameTemplate)
	if err != nil {
		return "", err
	}
	err = t.Execute(&out, data)
	return out.String(), err
}
