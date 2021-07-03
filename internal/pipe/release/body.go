package release

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const bodyTemplateText = `{{ with .Header }}{{ . }}{{ "\n" }}{{ end }}
{{- .ReleaseNotes }}

{{- with .DockerImages }}

## Docker images
{{ range $element := . }}
- ` + "`docker pull {{ . -}}`" + `
{{- end -}}
{{- end }}
{{- with .Footer }}{{ "\n" }}{{ . }}{{ end }}
`

func isLatest(img string) bool {
	return strings.HasSuffix(img, ":latest") || !strings.Contains(img, ":")
}

func describeBody(ctx *context.Context) (bytes.Buffer, error) {
	var out bytes.Buffer
	// nolint:prealloc
	var dockers []string
	for _, a := range ctx.Artifacts.Filter(artifact.ByType(artifact.DockerManifest)).List() {
		if isLatest(a.Name) {
			continue
		}
		dockers = append(dockers, a.Name)
	}
	if len(dockers) == 0 {
		for _, a := range ctx.Artifacts.Filter(artifact.ByType(artifact.DockerImage)).List() {
			if isLatest(a.Name) {
				continue
			}
			dockers = append(dockers, a.Name)
		}
	}

	header, err := tmpl.New(ctx).Apply(ctx.Config.Release.Header)
	if err != nil {
		return out, err
	}
	footer, err := tmpl.New(ctx).Apply(ctx.Config.Release.Footer)
	if err != nil {
		return out, err
	}

	bodyTemplate := template.Must(template.New("release").Parse(bodyTemplateText))
	err = bodyTemplate.Execute(&out, struct {
		Header       string
		Footer       string
		ReleaseNotes string
		DockerImages []string
	}{
		Header:       header,
		Footer:       footer,
		ReleaseNotes: ctx.ReleaseNotes,
		DockerImages: dockers,
	})
	return out, err
}
