package release

import (
	"bytes"
	"os"
	"text/template"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const bodyTemplateText = `{{ with .Header }}{{ . }}{{ "\n" }}{{ end }}
{{- .ReleaseNotes }}
{{- with .Footer }}{{ "\n" }}{{ . }}{{ end }}
`

func describeBody(ctx *context.Context) (bytes.Buffer, error) {
	var out bytes.Buffer
	f := tmpl.Fields{}
	var checksum string
	checksums := map[string]string{}
	checks, err := ctx.Artifacts.Checksums().OnlyChecksums()
	if err != nil {
		return out, err
	}
	for _, check := range checks {
		bts, err := os.ReadFile(check.Path)
		if err != nil {
			return out, err
		}
		checksum = string(bts)
		of := artifact.ExtraOr(*check, artifact.ExtraChecksumOf, "<unknown>")
		checksums[of] = string(bts)
	}
	if len(checks) == 1 {
		f["Checksums"] = checksum
	} else {
		f["Checksums"] = checksums
	}

	t := tmpl.New(ctx).WithExtraFields(f)

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
