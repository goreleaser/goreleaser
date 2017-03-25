package build

import (
	"bytes"
	"log"
	"text/template"
	"time"

	"github.com/goreleaser/goreleaser/context"
)

type ldflagsData struct {
	Date    string
	Commit  string
	Version string
}

func ldflags(ctx *context.Context) (string, error) {
	var data = ldflagsData{
		Commit:  ctx.Git.Commit,
		Version: ctx.Git.CurrentTag,
		Date:    time.Now().Format("2006-01-02_15:04:05"),
	}
	log.Println(ctx.Git)
	var out bytes.Buffer
	t, err := template.New("ldflags").Parse(ctx.Config.Build.Ldflags)
	if err != nil {
		return "", err
	}
	err = t.Execute(&out, data)
	return out.String(), err
}
