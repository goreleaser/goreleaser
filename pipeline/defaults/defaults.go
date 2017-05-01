// Package defaults implements the Pipe interface providing default values
// for missing configuration.
package defaults

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/goreleaser/goreleaser/context"
)

var defaultFiles = []string{"licence", "license", "readme", "changelog"}

// NameTemplate default name_template for the archive.
const NameTemplate = "{{ .Binary }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"

// SnapshotNameTemplate represents the default format for snapshot release names.
const SnapshotNameTemplate = "SNAPSHOT-{{ .Commit }}"

// Pipe for brew deployment
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Setting defaults"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	ctx.Config.Dist = "dist"
	if ctx.Config.Snapshot.NameTemplate == "" {
		ctx.Config.Snapshot.NameTemplate = SnapshotNameTemplate
	}
	if err := setReleaseDefaults(ctx); err != nil {
		return err
	}
	setBuildDefaults(ctx)
	if ctx.Config.Brew.Install == "" {
		ctx.Config.Brew.Install = fmt.Sprintf(
			`bin.install "%s"`,
			ctx.Config.Build.Binary,
		)
	}
	return setArchiveDefaults(ctx)
}

func setReleaseDefaults(ctx *context.Context) error {
	if ctx.Config.Release.GitHub.Name != "" {
		return nil
	}
	repo, err := remoteRepo()
	if err != nil {
		return fmt.Errorf("failed reading repo from git: %v", err.Error())
	}
	ctx.Config.Release.GitHub = repo
	return nil
}

func setBuildDefaults(ctx *context.Context) {
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
	if len(ctx.Config.Build.Goarm) == 0 {
		ctx.Config.Build.Goarm = []string{"6"}
	}
	if ctx.Config.Build.Ldflags == "" {
		ctx.Config.Build.Ldflags = "-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}"
	}
}

func setArchiveDefaults(ctx *context.Context) error {
	if ctx.Config.Archive.NameTemplate == "" {
		ctx.Config.Archive.NameTemplate = NameTemplate
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
