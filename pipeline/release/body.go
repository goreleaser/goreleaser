package release

import (
	"bytes"
	"text/template"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
)

const bodyTemplateText = `{{ .ReleaseNotes }}

{{- with .DockerImages }}

## Docker images
{{ range $element := . }}
- ` + "`docker pull {{ . -}}`" + `
{{- end -}}
{{- end }}
`

var bodyTemplate *template.Template

func init() {
	bodyTemplate = template.Must(template.New("release").Parse(bodyTemplateText))
}

func describeBody(ctx *context.Context) (bytes.Buffer, error) {
	var out bytes.Buffer
	// nolint:prealloc
	var dockers []string
	for _, a := range ctx.Artifacts.Filter(artifact.ByType(artifact.DockerImage)).List() {
		dockers = append(dockers, a.Name)
	}
	err := bodyTemplate.Execute(&out, struct {
		ReleaseNotes string
		DockerImages []string
	}{
		ReleaseNotes: ctx.ReleaseNotes,
		DockerImages: dockers,
	})
	return out, err
}
