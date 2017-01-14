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
func (Pipe) Run(ctx *context.Context) (err error) {
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
		return
	}
	ctx.Config.Files = []string{}
	for _, pattern := range filePatterns {
		matches, err := globPath(pattern)
		if err != nil {
			return err
		}

		ctx.Config.Files = append(ctx.Config.Files, matches...)
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
