package client

import (
	"bytes"
	"text/template"

	"github.com/goreleaser/goreleaser/context"
)

func releaseTitle(ctx *context.Context) (string, error) {
	var out bytes.Buffer
	t, err := template.New("github").Parse(ctx.Config.Release.NameTemplate)
	if err != nil {
		return "", err
	}
	err = t.Execute(&out, struct {
		ProjectName, Tag, Version string
	}{
		ProjectName: ctx.Config.ProjectName,
		Tag:         ctx.Git.CurrentTag,
		Version:     ctx.Version,
	})
	return out.String(), err
}
