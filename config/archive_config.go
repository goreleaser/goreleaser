package config

import (
	"bytes"
	"html/template"

	"github.com/goreleaser/releaser/uname"
)

// ArchiveConfig config used for the archive
type ArchiveConfig struct {
	Format       string
	NameTemplate string `yaml:"name_template"`
}

type archiveNameData struct {
	Os         string
	Arch       string
	Version    string
	BinaryName string
}

// ArchiveName following the given template
func (config ProjectConfig) ArchiveName(goos, goarch string) (string, error) {
	var data = archiveNameData{
		Os:         uname.FromGo(goos),
		Arch:       uname.FromGo(goarch),
		Version:    config.Git.CurrentTag,
		BinaryName: config.BinaryName,
	}
	var out bytes.Buffer
	t, err := template.New(data.BinaryName).Parse(config.Archive.NameTemplate)
	if err != nil {
		return "", err
	}
	err = t.Execute(&out, data)
	return out.String(), err
}
