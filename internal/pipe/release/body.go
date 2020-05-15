package release

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const bodyTemplateText = `{{if .Header }}{{ .Header }}

{{ .ReleaseNotes }}{{else}}{{ .ReleaseNotes }}{{end}}

{{- with .DockerImages }}

## Docker images
{{ range $element := . }}
- ` + "`docker pull {{ . -}}`" + `
{{- end -}}
{{- end }}
{{if .Footer }}
{{ .Footer }}
{{end}}`

func describeBody(ctx *context.Context) (bytes.Buffer, error) {
	var out bytes.Buffer
	// nolint:prealloc
	var dockers []string

	for _, a := range ctx.Artifacts.Filter(artifact.ByType(artifact.DockerImage)).List() {
		dockers = append(dockers, a.Name)
	}

	h, err := describeTemplate(ctx, ctx.Config.Release.HeaderTemplate)
	if err != nil {
		return out, err
	}

	f, err := describeTemplate(ctx, ctx.Config.Release.FooterTemplate)
	if err != nil {
		return out, err
	}

	var bodyTemplate = template.Must(template.New("release").Parse(bodyTemplateText))
	err = bodyTemplate.Execute(&out, struct {
		Header       string
		ReleaseNotes string
		DockerImages []string
		Footer       string
	}{
		ReleaseNotes: ctx.ReleaseNotes,
		DockerImages: dockers,
		Header:       h,
		Footer:       f,
	})

	return out, err
}

func describeTemplate(ctx *context.Context, template string) (string, error) {
	if template == "" {
		return "", nil
	}

	t, err := tmpl.New(ctx).Apply(template)

	fmt.Println(t)
	return t, err
}
