package nametemplate

import (
	"bytes"
	"text/template"
	"time"

	"github.com/goreleaser/goreleaser/context"
)

func Apply(ctx *context.Context, name, tmpl string) (string, error) {
	var out bytes.Buffer
	t, err := template.New(name).
		Option("missingkey=error").
		Funcs(mkFuncMap()).
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

func mkFuncMap() template.FuncMap {
	return template.FuncMap{
		"time": func(s string) string {
			return time.Now().UTC().Format(s)
		},
	}
}
