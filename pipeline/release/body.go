package release

import (
	"bytes"
	"os/exec"
	"text/template"

	"github.com/goreleaser/goreleaser/context"
)

const bodyTemplate = `{{ .ReleaseNotes }}

{{- if .DockerImages }}

Docker images:
{{ range $element := .DockerImages }}
- {{ . -}}
{{ end -}}
{{- end }}

---
Automated with [GoReleaser](https://github.com/goreleaser)
Built with {{ .GoVersion }}`

func describeBody(ctx *context.Context) (bytes.Buffer, error) {
	bts, err := exec.Command("go", "version").CombinedOutput()
	if err != nil {
		return bytes.Buffer{}, err
	}
	return describeBodyVersion(ctx, string(bts))
}

func describeBodyVersion(ctx *context.Context, version string) (bytes.Buffer, error) {
	var out bytes.Buffer
	var template = template.Must(template.New("release").Parse(bodyTemplate))
	err := template.Execute(&out, struct {
		ReleaseNotes, GoVersion string
		DockerImages            []string
	}{
		ReleaseNotes: ctx.ReleaseNotes,
		GoVersion:    version,
		DockerImages: ctx.Dockers,
	})
	return out, err
}
