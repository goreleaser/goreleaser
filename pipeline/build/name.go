package build

import (
	"bytes"
	"log"
	"strings"
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

	// TODO: remove this block in next release cycle
	if strings.Contains(ctx.Config.Archive.NameTemplate, ".BinaryName") {
		log.Println("The `.BinaryName` in `archive.name_template` is deprecated and will soon be removed. Please check the README for more info.")
		ctx.Config.Archive.NameTemplate = strings.Replace(
			ctx.Config.Archive.NameTemplate,
			".BinaryName",
			".Binary",
			-1,
		)
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
