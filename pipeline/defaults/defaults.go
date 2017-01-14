package defaults

import (
	"os"
	"path"
	"path/filepath"

	"github.com/goreleaser/releaser/context"
)

var filePatterns = []string{"LICENCE*", "LICENSE*", "README*", "CHANGELOG*"}

// Pipe for brew deployment
type Pipe struct{}

// Name of the pipe
func (Pipe) Name() string {
	return "Defaults"
}

// Run the pipe
func (Pipe) Run(context *context.Context) (err error) {
	if context.Config.Build.Main == "" {
		context.Config.Build.Main = "main.go"
	}
	if len(context.Config.Build.Oses) == 0 {
		context.Config.Build.Oses = []string{"linux", "darwin"}
	}
	if len(context.Config.Build.Arches) == 0 {
		context.Config.Build.Arches = []string{"amd64", "386"}
	}
	if context.Config.Build.Ldflags == "" {
		context.Config.Build.Ldflags = "-s -w"
	}
	if context.Config.Archive.NameTemplate == "" {
		context.Config.Archive.NameTemplate = "{{.BinaryName}}_{{.Os}}_{{.Arch}}"
	}
	if context.Config.Archive.Format == "" {
		context.Config.Archive.Format = "tar.gz"
	}
	if len(context.Config.Archive.Replacements) == 0 {
		context.Config.Archive.Replacements = map[string]string{
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
	if len(context.Config.Files) != 0 {
		return
	}
	context.Config.Files = []string{}
	for _, pattern := range filePatterns {
		matches, err := globPath(pattern)
		if err != nil {
			return err
		}

		context.Config.Files = append(context.Config.Files, matches...)
	}
	return
}

func globPath(p string) (m []string, err error) {
	var cwd string
	var dirs []string

	if cwd, err = os.Getwd(); err != nil {
		return
	}

	fp := path.Join(cwd, p)

	if dirs, err = filepath.Glob(fp); err != nil {
		return
	}

	// Normalise to avoid nested dirs in tarball
	for _, dir := range dirs {
		_, f := filepath.Split(dir)
		m = append(m, f)
	}

	return
}
