package build

import (
	"bytes"
	"text/template"

	"github.com/goreleaser/goreleaser/config"
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

func nameFor(ctx *context.Context, build config.Build, target buildTarget) (string, error) {
	var data = nameData{
		Os:      replace(ctx.Config.Archive.Replacements, target.goos),
		Arch:    replace(ctx.Config.Archive.Replacements, target.goarch),
		Arm:     replace(ctx.Config.Archive.Replacements, target.goarm),
		Version: ctx.Version,
		Tag:     ctx.Git.CurrentTag,
		Binary:  build.Binary,
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
