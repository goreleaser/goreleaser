// Package sourcearchive archives the source of the project using git-archive.
package sourcearchive

import (
	"path/filepath"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/archivefiles"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe for source archive.
type Pipe struct{}

func (Pipe) String() string {
	return "creating source archive"
}

func (Pipe) Skip(ctx *context.Context) bool {
	return !ctx.Config.Source.Enabled
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) (err error) {
	name, err := tmpl.New(ctx).Apply(ctx.Config.Source.NameTemplate)
	if err != nil {
		return err
	}
	filename := name + "." + ctx.Config.Source.Format
	path := filepath.Join(ctx.Config.Dist, filename)
	log.WithField("file", filename).Info("creating source archive")
	args := []string{
		"archive",
		"-o", path,
		"--format", ctx.Config.Source.Format,
	}

	tpl := tmpl.New(ctx)
	prefix, err := tpl.Apply(ctx.Config.Source.PrefixTemplate)
	if err != nil {
		return err
	}
	if prefix != "" {
		args = append(args, "--prefix", prefix)
	}

	var files []config.File
	for _, f := range ctx.Config.Source.Files {
		files = append(files, config.File{
			Source: f,
		})
	}
	addFiles, err := archivefiles.Eval(tpl, files)
	if err != nil {
		return err
	}

	for _, f := range addFiles {
		if isTracked(ctx, f.Source) {
			continue
		}
		args = append(args, "--add-file", f.Source)
	}

	args = append(args, ctx.Git.FullCommit)
	out, err := git.Clean(git.Run(ctx, args...))
	log.Debug(out)

	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableSourceArchive,
		Name: filename,
		Path: path,
		Extra: map[string]interface{}{
			artifact.ExtraFormat: ctx.Config.Source.Format,
		},
	})
	return err
}

func isTracked(ctx *context.Context, path string) bool {
	_, err := git.Run(ctx, "ls-files", "--error-unmatch", path)
	return err == nil
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	archive := &ctx.Config.Source
	if archive.Format == "" {
		archive.Format = "tar.gz"
	}

	if archive.NameTemplate == "" {
		archive.NameTemplate = "{{ .ProjectName }}-{{ .Version }}"
	}
	return nil
}
