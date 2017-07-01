package name

import (
	"bytes"
	"text/template"

	"github.com/goreleaser/goreleaser/context"
)

type nameData struct {
	Os      string
	Arch    string
	Arm     string
	Version string
	Tag     string
	Binary  string
}

func For(ctx *context.Context, goos, goarch, goarm string) (string, error) {
	var data = nameData{
		Os:      replace(ctx.Config.Archive.Replacements, goos),
		Arch:    replace(ctx.Config.Archive.Replacements, goarch),
		Arm:     replace(ctx.Config.Archive.Replacements, goarm),
		Version: ctx.Version,
		Tag:     ctx.Git.CurrentTag,
		Binary:  ctx.Config.Name,
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
