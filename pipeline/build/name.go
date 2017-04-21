package build

import (
	"bytes"
	"text/template"

	"github.com/goreleaser/goreleaser/context"
)

type nameData struct {
	Os      string
	Arch    string
	Version string
	Binary  string
}

func nameFor(ctx *context.Context, goos, goarch string) (string, error) {
	var data = nameData{
		Os:      replace(ctx.Config.Archive.Replacements, goos),
		Arch:    replace(ctx.Config.Archive.Replacements, goarch),
		Version: ctx.Git.CurrentTag,
		Binary:  ctx.Config.Build.Binary,
	}

	var out bytes.Buffer
	t, err := template.New(data.Binary).Parse(ctx.Config.Archive.NameTemplate)
	if err != nil {
		return "", err
	}
	err = t.Execute(&out, data)
	return out.String(), err
}

func replace(replacements map[string]string, original string) string {
	result := replacements[original]
	if result == "" {
		return original
	}
	return result
}

func extFor(goos string) string {
	if goos == "windows" {
		return ".exe"
	}
	return ""
}
