package defaults

import (
	"errors"
	"io/ioutil"
	"strings"

	"github.com/goreleaser/releaser/context"
)

var defaultFiles = []string{"licence", "license", "readme", "changelog"}

// Pipe for brew deployment
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Setting defaults..."
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	if ctx.Config.Repo == "" {
		repo, err := remoteRepo()
		ctx.Config.Repo = repo
		if err != nil {
			return errors.New("failed reading repo from Git: " + err.Error())
		}
	}
	if ctx.Config.BinaryName == "" {
		ctx.Config.BinaryName = strings.Split(ctx.Config.Repo, "/")[1]
	}
	if ctx.Config.Build.Main == "" {
		ctx.Config.Build.Main = "main.go"
	}
	if len(ctx.Config.Build.Oses) == 0 {
		ctx.Config.Build.Oses = []string{"linux", "darwin"}
	}
	if len(ctx.Config.Build.Arches) == 0 {
		ctx.Config.Build.Arches = []string{"amd64", "386"}
	}
	if ctx.Config.Build.Ldflags == "" {
		ctx.Config.Build.Ldflags = "-s -w"
	}
	if ctx.Config.Archive.NameTemplate == "" {
		ctx.Config.Archive.NameTemplate = "{{.BinaryName}}_{{.Os}}_{{.Arch}}"
	}
	if ctx.Config.Archive.Format == "" {
		ctx.Config.Archive.Format = "tar.gz"
	}
	if len(ctx.Config.Archive.Replacements) == 0 {
		ctx.Config.Archive.Replacements = map[string]string{
			"darwin":  "Darwin",
			"linux":   "Linux",
			"freebsd": "FreeBSD",
			"openbsd": "OpenBSD",
			"netbsd":  "NetBSD",
			"windows": "Windows",
			"386":     "i386",
			"amd64":   "x86_64",
		}
	}
	if len(ctx.Config.Files) != 0 {
		return nil
	}
	files, err := findFiles()
	if err != nil {
		return err
	}
	ctx.Config.Files = files
	return nil
}

func findFiles() (files []string, err error) {
	all, err := ioutil.ReadDir(".")
	if err != nil {
		return
	}
	for _, file := range all {
		if accept(file.Name()) {
			files = append(files, file.Name())
		}
	}
	return
}

func accept(file string) bool {
	for _, accepted := range defaultFiles {
		if strings.HasPrefix(strings.ToLower(file), accepted) {
			return true
		}
	}
	return false
}
