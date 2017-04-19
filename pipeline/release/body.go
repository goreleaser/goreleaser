package release

import (
	"bytes"
	"html/template"
	"os/exec"

	"github.com/goreleaser/goreleaser/context"
)

const bodyTemplate = `## Changelog

{{ .Changelog }}

---
Automated with @goreleaser
Built with {{ .GoVersion }}
`

func buildBody(ctx *context.Context) (bytes.Buffer, error) {
	var out bytes.Buffer
	bts, err := exec.Command("go", "version").CombinedOutput()
	if err != nil {
		return out, err
	}
	var template = template.Must(template.New("release").Parse(bodyTemplate))
	err = template.Execute(&out, struct {
		Changelog, GoVersion string
	}{
		Changelog: ctx.Changelog,
		GoVersion: string(bts),
	})
	return out, err
}
