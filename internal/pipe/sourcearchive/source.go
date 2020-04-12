// Package sourcearchive archives the source of the project using git-archive.
package sourcearchive

import (
	"path/filepath"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe for cleandis
type Pipe struct{}

func (Pipe) String() string {
	return "creating source archive"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) (err error) {
	if !ctx.Config.Source.Enabled {
		return pipe.Skip("source pipe is disabled")
	}

	name, err := tmpl.New(ctx).Apply(ctx.Config.Source.NameTemplate)
	if err != nil {
		return err
	}
	var filename = name + "." + ctx.Config.Source.Format
	var path = filepath.Join(ctx.Config.Dist, filename)
	log.WithField("file", filename).Info("creating source archive")
	out, err := git.Clean(git.Run("archive", "-o", path, ctx.Git.FullCommit))
	log.Debug(out)
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableSourceArchive,
		Name: filename,
		Path: path,
		Extra: map[string]interface{}{
			"Format": ctx.Config.Source.Format,
		},
	})
	return err
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	var archive = &ctx.Config.Source
	if archive.Format == "" {
		archive.Format = "tar.gz"
	}

	if archive.NameTemplate == "" {
		archive.NameTemplate = "{{ .ProjectName }}-{{ .Version }}"
	}
	return nil
}
