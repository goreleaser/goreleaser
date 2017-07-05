package release

import (
	"bytes"
	"os/exec"
	"text/template"

	"github.com/goreleaser/goreleaser/context"
)

const bodyTemplate = `{{ .ReleaseNotes }}

---
Automated with [GoReleaser](https://github.com/goreleaser)
Built with {{ .GoVersion }}
`

func describeBody(ctx *context.Context) (bytes.Buffer, error) {
	var out bytes.Buffer
	bts, err := exec.Command("go", "version").CombinedOutput()
	if err != nil {
		return out, err
	}
	var template = template.Must(template.New("release").Parse(bodyTemplate))
	err = template.Execute(&out, struct {
		ReleaseNotes, GoVersion string
	}{
		ReleaseNotes: ctx.ReleaseNotes,
		GoVersion:    string(bts),
	})
	return out, err
}
