// Package defaults implements the Pipe interface providing default values
// for missing configuration.
package defaults

import (
	"errors"
	"io/ioutil"
	"strings"

	"github.com/goreleaser/goreleaser/context"
)

var defaultFiles = []string{"licence", "license", "readme", "changelog"}

// Pipe for brew deployment
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Setting defaults"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	if ctx.Config.Release.GitHub.Name == "" {
		repo, err := remoteRepo()
		ctx.Config.Release.GitHub = repo
		if err != nil {
			return errors.New("failed reading repo from git: " + err.Error())
		}
	}
	if ctx.Config.Build.Binary == "" {
		ctx.Config.Build.Binary = ctx.Config.Release.GitHub.Name
	}
	if ctx.Config.Build.Main == "" {
		ctx.Config.Build.Main = "."
	}
	if len(ctx.Config.Build.Goos) == 0 {
		ctx.Config.Build.Goos = []string{"linux", "darwin"}
	}
	if len(ctx.Config.Build.Goarch) == 0 {
		ctx.Config.Build.Goarch = []string{"amd64", "386"}
	}
	if ctx.Config.Build.Ldflags == "" {
		ctx.Config.Build.Ldflags = "-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}"
	}

	if ctx.Config.Archive.NameTemplate == "" {
		ctx.Config.Archive.NameTemplate = "{{.Binary}}_{{.Os}}_{{.Arch}}"
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
	if len(ctx.Config.Archive.Files) == 0 {
		files, err := findFiles()
		if err != nil {
			return err
		}
		ctx.Config.Archive.Files = files
	}
	if ctx.Config.Brew.Install == "" {
		ctx.Config.Brew.Install = "bin.install \"" + ctx.Config.Build.Binary + "\""
	}
	ctx.Config.Dist = "dist"
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
