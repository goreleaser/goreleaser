// Package sourcearchive archives the source of the project using git-archive.
package sourcearchive

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/archivefiles"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/deprecate"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/archive"
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
func (Pipe) Run(ctx *context.Context) error {
	format := ctx.Config.Source.Format
	if format != "zip" && format != "tar" && format != "tgz" && format != "tar.gz" {
		return fmt.Errorf("invalid source archive format: %s", format)
	}
	name, err := tmpl.New(ctx).Apply(ctx.Config.Source.NameTemplate)
	if err != nil {
		return err
	}
	filename := name + "." + format
	path := filepath.Join(ctx.Config.Dist, filename)
	log.WithField("file", filename).Info("creating source archive")
	args := []string{
		"archive",
		"-o", path,
	}

	prefix := ""
	if ctx.Config.Source.PrefixTemplate != "" {
		pt, err := tmpl.New(ctx).Apply(ctx.Config.Source.PrefixTemplate)
		if err != nil {
			return err
		}
		prefix = pt
		args = append(args, "--prefix", prefix)
	}
	args = append(args, ctx.Git.FullCommit)

	if _, err := git.Clean(git.Run(ctx, args...)); err != nil {
		return err
	}

	if len(ctx.Config.Source.Files) > 0 {
		if err := appendExtraFilesToArchive(ctx, prefix, path, format); err != nil {
			return err
		}
	}

	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableSourceArchive,
		Name: filename,
		Path: path,
		Extra: map[string]interface{}{
			artifact.ExtraFormat: format,
		},
	})
	return err
}

func appendExtraFilesToArchive(ctx *context.Context, prefix, path, format string) error {
	oldPath := path + ".bkp"
	if err := gio.Copy(path, oldPath); err != nil {
		return fmt.Errorf("failed make a backup of %q: %w", path, err)
	}

	// i could spend a lot of time trying to figure out how to append to a tar,
	// tgz and zip file... but... this seems easy enough :)
	of, err := os.Open(oldPath)
	if err != nil {
		return fmt.Errorf("could not open %q: %w", oldPath, err)
	}
	defer of.Close()

	af, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("could not open archive: %w", err)
	}
	defer af.Close()

	arch, err := archive.Copying(of, af, format)
	if err != nil {
		return err
	}

	files, err := archivefiles.Eval(tmpl.New(ctx), ctx.Config.Source.Files)
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
	return nil
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

	if archive.Enabled && archive.RLCP != "" {
		deprecate.Notice(ctx, "source.rlcp")
	}
	return nil
}
