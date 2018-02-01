package release

import (
	"bytes"
	"os/exec"
	"text/template"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
)

const bodyTemplateText = `{{ .ReleaseNotes }}

{{- if .DockerImages }}

## Docker images
{{ range $element := .DockerImages }}
- ` + "`docker pull {{ . -}}`" + `
{{- end -}}
{{- end }}

---
Automated with [GoReleaser](https://github.com/goreleaser)
Built with {{ .GoVersion }}`

var bodyTemplate *template.Template

func init() {
	bodyTemplate = template.Must(template.New("release").Parse(bodyTemplateText))
}

func describeBody(ctx *context.Context) (bytes.Buffer, error) {
	/* #nosec */
	bts, err := exec.CommandContext(ctx, "go", "version").CombinedOutput()
	if err != nil {
		return bytes.Buffer{}, err
	}
	return describeBodyVersion(ctx, string(bts))
}

func describeBodyVersion(ctx *context.Context, version string) (bytes.Buffer, error) {
	var out bytes.Buffer
	var dockers []string
	for _, a := range ctx.Artifacts.Filter(artifact.ByType(artifact.DockerImage)).List() {
		dockers = append(dockers, a.Name)
	}
	err := bodyTemplate.Execute(&out, struct {
		ReleaseNotes, GoVersion string
		DockerImages            []string
	}{
		ReleaseNotes: ctx.ReleaseNotes,
		GoVersion:    version,
		DockerImages: dockers,
	})
	return out, err
}
