package release

import (
	"bytes"
	"text/template"

	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const bodyTemplateText = `{{ with .Header }}{{ . }}{{ "\n" }}{{ end }}
{{- .ReleaseNotes }}
{{- with .Footer }}{{ "\n" }}{{ . }}{{ end }}
`

func describeBody(ctx *context.Context) (bytes.Buffer, error) {
	var out bytes.Buffer
	t := tmpl.New(ctx)

	header, err := t.Apply(ctx.Config.Release.Header)
	if err != nil {
		return out, err
	}
	footer, err := t.Apply(ctx.Config.Release.Footer)
	if err != nil {
		return out, err
	}

	bodyTemplate := template.Must(template.New("release").Parse(bodyTemplateText))
	err = bodyTemplate.Execute(&out, struct {
		Header       string
		Footer       string
		ReleaseNotes string
	}{
		Header:       header,
		Footer:       footer,
		ReleaseNotes: ctx.ReleaseNotes,
	})
	return out, err
}
