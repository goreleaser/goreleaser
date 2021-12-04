// Package sourcearchive archives the source of the project using git-archive.
package sourcearchive

import (
	"path/filepath"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/tmpl"
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
	}
	if ctx.Config.Source.PrefixTemplate != "" {
		prefix, err := tmpl.New(ctx).Apply(ctx.Config.Source.PrefixTemplate)
		if err != nil {
			return err
		}
		args = append(args, "--prefix", prefix)
	}
	args = append(args, ctx.Git.FullCommit)
	out, err := git.Clean(git.Run(args...))
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
