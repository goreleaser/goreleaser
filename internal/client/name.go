package client

import (
	"bytes"
	"text/template"
	"time"

	"github.com/goreleaser/goreleaser/context"
)

var (
	timeNow = time.Now
)

func releaseTitle(ctx *context.Context) (string, error) {
	var out bytes.Buffer
	t, err := template.New("github").
		Option("missingkey=error").
		Funcs(mkFuncMap()).
		Parse(ctx.Config.Release.NameTemplate)
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
		"time": func(s ...string) (string, error) {
			if len(s) < 1 {
				return "", nil
			}
			return timeNow().Format(s[0]), nil
		},
	}
}
