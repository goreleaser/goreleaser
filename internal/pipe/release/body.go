package release

import (
	"bytes"
	"text/template"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const bodyTemplateText = `{{ .ReleaseNotes }}

{{- with .DockerImages }}

## Docker images
{{ range $element := . }}
- ` + "`docker pull {{ . -}}`" + `
{{- end -}}
{{- end }}
`

func describeBody(ctx *context.Context) (bytes.Buffer, error) {
	var out bytes.Buffer

	h, err := describeTemplate(ctx, ctx.Config.Release.HeaderTemplate)
	if err != nil {
		return out, err
	}
	out.WriteString(h)

	b, err := mountBody(ctx)
	if err != nil {
		return out, err
	}
	out.Write(b)

	f, err := describeTemplate(ctx, ctx.Config.Release.FooterTemplate)
	if err != nil {
		return out, err
	}
	out.WriteString(f)

	return out, nil
}

func mountBody(ctx *context.Context) ([]byte, error) {
	var out bytes.Buffer
	// nolint:prealloc
	var dockers []string
	for _, a := range ctx.Artifacts.Filter(artifact.ByType(artifact.DockerImage)).List() {
		dockers = append(dockers, a.Name)
	}
	var bodyTemplate = template.Must(template.New("release").Parse(bodyTemplateText))
	err := bodyTemplate.Execute(&out, struct {
		ReleaseNotes string
		DockerImages []string
	}{
		ReleaseNotes: ctx.ReleaseNotes,
		DockerImages: dockers,
	})

	return out.Bytes(), err
}

func describeTemplate(ctx *context.Context, template string) (string, error) {
	if template == "" {
		return "", nil
	}

	h, err := tmpl.New(ctx).Apply(template)

	return h, err
}
