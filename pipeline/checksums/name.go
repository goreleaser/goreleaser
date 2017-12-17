package checksums

import (
	"bytes"
	"text/template"

	"github.com/goreleaser/goreleaser/context"
)

func filenameFor(ctx *context.Context) (string, error) {
	var out bytes.Buffer
	t, err := template.New("checksums").Parse(ctx.Config.Checksum.NameTemplate)
	if err != nil {
		return "", err
	}
	err = t.Execute(&out, struct {
		ProjectName string
		Tag         string
		Version     string
		Env         map[string]string
	}{
		ProjectName: ctx.Config.ProjectName,
		Tag:         ctx.Git.CurrentTag,
		Version:     ctx.Version,
		Env:         ctx.Env,
	})
	return out.String(), err
}
