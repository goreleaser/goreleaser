// Package nametemplate provides common template function for releases and etc.
package nametemplate

import (
	"bytes"
	"text/template"
	"time"

	"github.com/goreleaser/goreleaser/context"
)

// Apply applies the given name template using the context as source.
func Apply(ctx *context.Context, tmpl string) (string, error) {
	var out bytes.Buffer
	t, err := template.New("release").
		Option("missingkey=error").
		Funcs(template.FuncMap{
			"time": func(s string) string {
				return time.Now().UTC().Format(s)
			},
		}).
		Parse(tmpl)
	if err != nil {
		return "", err
	}
	err = t.Execute(&out, struct {
		ProjectName, Tag, Version string
	}{
		ProjectName: ctx.Config.ProjectName,
		Tag:         ctx.Git.CurrentTag,
		Version:     ctx.Version,
	})
	return out.String(), err
}
