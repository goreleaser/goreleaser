package build

import (
	"bytes"
	"text/template"
	"time"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
)

type ldflagsData struct {
	Date    string
	Tag     string
	Commit  string
	Version string
}

func ldflags(ctx *context.Context, build config.Build) (string, error) {
	var data = ldflagsData{
		Commit:  ctx.Git.Commit,
		Tag:     ctx.Git.CurrentTag,
		Version: ctx.Version,
		Date:    time.Now().UTC().Format(time.RFC3339),
	}
	var out bytes.Buffer
	t, err := template.New("ldflags").Parse(build.Ldflags)
	if err != nil {
		return "", err
	}
	err = t.Execute(&out, data)
	return out.String(), err
}
