// Package nametemplate provides common template function for releases and etc.
package nametemplate

import (
	"bytes"
	"text/template"
	"time"

	"github.com/goreleaser/goreleaser/context"
	"github.com/masterminds/semver"
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
	sv, err := semver.NewVersion(ctx.Git.CurrentTag)
	if err != nil {
		return "", err
	}
	err = t.Execute(&out, struct {
		ProjectName string
		Tag         string
		Version     string
		Commit      string
		Major       int64
		Minor       int64
		Patch       int64
		Env         map[string]string
	}{
		ProjectName: ctx.Config.ProjectName,
		Tag:         ctx.Git.CurrentTag,
		Version:     ctx.Version,
		Commit:      ctx.Git.Commit,
		Major:       sv.Major(),
		Minor:       sv.Minor(),
		Patch:       sv.Patch(),
		Env:         ctx.Env,
	})
	return out.String(), err
}
