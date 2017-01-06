package name

import (
	"bytes"
	"text/template"

	"github.com/goreleaser/releaser/config"
	"github.com/goreleaser/releaser/uname"
)

type nameData struct {
	Os         string
	Arch       string
	Version    string
	BinaryName string
}

func For(config config.ProjectConfig, goos, goarch string) (string, error) {
	var data = nameData{
		Os:         uname.FromGo(goos),
		Arch:       uname.FromGo(goarch),
		Version:    config.Git.CurrentTag,
		BinaryName: config.BinaryName,
	}
	var out bytes.Buffer
	template, err := template.New(data.BinaryName).Parse(config.NameTemplate)
	if err != nil {
		return "", err
	}
	err = template.Execute(&out, data)
	return out.String(), err
}
