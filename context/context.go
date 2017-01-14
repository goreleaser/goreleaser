package context

import (
	"bytes"
	"html/template"

	"github.com/goreleaser/releaser/config"
)

// GitInfo includes tags and diffs used in some point
type GitInfo struct {
	CurrentTag  string
	PreviousTag string
	Diff        string
}

type Repo struct {
	Owner, Name string
}

type Context struct {
	Config   *config.ProjectConfig
	Token    *string
	Git      *GitInfo
	Repo     *Repo
	BrewRepo *Repo
	Archives []string
}

func New(config config.ProjectConfig) *Context {
	return &Context{
		Config: &config,
	}
}

type archiveNameData struct {
	Os         string
	Arch       string
	Version    string
	BinaryName string
}

// ArchiveName
func (context *Context) ArchiveName(goos, goarch string) (string, error) {
	var data = archiveNameData{
		Os:         replace(context.Config.Archive.Replacements, goos),
		Arch:       replace(context.Config.Archive.Replacements, goarch),
		Version:    context.Git.CurrentTag,
		BinaryName: context.Config.BinaryName,
	}
	var out bytes.Buffer
	t, err := template.New(data.BinaryName).Parse(context.Config.Archive.NameTemplate)
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
