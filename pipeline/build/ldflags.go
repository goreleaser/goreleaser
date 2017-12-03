package build

import (
	"bytes"
	"os"
	"strings"
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
	Env     map[string]string
}

func ldflags(ctx *context.Context, build config.Build) (string, error) {
	var data = ldflagsData{
		Commit:  ctx.Git.Commit,
		Tag:     ctx.Git.CurrentTag,
		Version: ctx.Version,
		Date:    time.Now().UTC().Format(time.RFC3339),
		Env:     loadEnvs(),
	}
	var out bytes.Buffer
	t, err := template.New("ldflags").Parse(build.Ldflags)
	if err != nil {
		return "", err
	}
	err = t.Execute(&out, data)
	return out.String(), err
}

func loadEnvs() map[string]string {
	r := map[string]string{}
	for _, e := range os.Environ() {
		env := strings.Split(e, "=")
		r[env[0]] = env[1]
	}
	return r
}
