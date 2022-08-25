// Package sourcearchive archives the source of the project using git-archive.
package sourcearchive

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/archivefiles"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/archive"
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

	out, err := git.Run(ctx, "ls-files")
	if err != nil {
		return fmt.Errorf("could not list source files: %w", err)
	}

	prefix, err := tmpl.New(ctx).Apply(ctx.Config.Source.PrefixTemplate)
	if err != nil {
		return err
	}

	af, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("could not create archive: %w", err)
	}
	defer af.Close() //nolint:errcheck

	arch, err := archive.New(af, ctx.Config.Source.Format)
	if err != nil {
		return err
	}

	var ff []config.File
	for _, f := range strings.Split(out, "\n") {
		if strings.TrimSpace(f) == "" {
			continue
		}
		ff = append(ff, config.File{
			Source: f,
		})
	}
	files, err := archivefiles.Eval(tmpl.New(ctx), append(ff, ctx.Config.Source.Files...))
	if err != nil {
		return err
	}
	for _, f := range files {
		f.Destination = filepath.Join(prefix, f.Destination)
		if err := arch.Add(f); err != nil {
			return fmt.Errorf("could not add %q to archive: %w", f.Source, err)
		}
	}

	if err := arch.Close(); err != nil {
		return fmt.Errorf("could not close archive file: %w", err)
	}
	if err := af.Close(); err != nil {
		return fmt.Errorf("could not close archive file: %w", err)
	}

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
