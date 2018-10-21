// Package archive implements the pipe interface with the intent of
// archiving and compressing the binaries, readme, and other artifacts. It
// also provides an Archive interface which represents an archiving format.
package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/apex/log"
	"github.com/campoy/unique"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/deprecate"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	zglob "github.com/mattn/go-zglob"
	"golang.org/x/sync/errgroup"
)

const (
	defaultNameTemplate       = "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"
	defaultBinaryNameTemplate = "{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"
)

// Pipe for archive
type Pipe struct{}

func (Pipe) String() string {
	return "creating archives"
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	if !reflect.DeepEqual(ctx.Config.OldArchive, config.Archive{}) {
		deprecate.Notice("archive")
		ctx.Config.Archives = append(ctx.Config.Archives, ctx.Config.OldArchive)
	}
	for i, archive := range ctx.Config.Archives {
		if archive.Format == "" {
			archive.Format = "tar.gz"
		}
		if len(archive.Files) == 0 {
			archive.Files = []string{
				"licence*",
				"LICENCE*",
				"license*",
				"LICENSE*",
				"readme*",
				"README*",
				"changelog*",
				"CHANGELOG*",
			}
		}
		if archive.NameTemplate == "" {
			archive.NameTemplate = defaultNameTemplate
			if archive.Format == "binary" {
				archive.NameTemplate = defaultBinaryNameTemplate
			}
		}
		ctx.Config.Archives[i] = archive
	}
	return nil
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	var g errgroup.Group
	var filtered = ctx.Artifacts.Filter(artifact.ByType(artifact.Binary))
	for group, artifacts := range filtered.GroupByPlatform() {
		log.Debugf("group %s has %d binaries", group, len(artifacts))
		artifacts := artifacts
		g.Go(func() error {
			if packageFormat(ctx, artifacts[0].Goos) == "binary" {
				return skip(ctx, artifacts)
			}
			return create(ctx, artifacts)
		})
	}
	return g.Wait()
}

func create(ctx *context.Context, archive config.Archive, binaries []artifact.Artifact) error {
	var format = packageFormat(ctx, archive, binaries[0].Goos)
	folder, err := tmpl.New(ctx).
		WithArtifact(binaries[0], archive.Replacements).
		Apply(archive.NameTemplate)
	if err != nil {
		return err
	}
	archivePath := filepath.Join(ctx.Config.Dist, folder+"."+format)
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %s", archivePath, err.Error())
	}
	defer archiveFile.Close() // nolint: errcheck
	var log = log.WithField("archive", archivePath)
	log.Info("creating")
	var a = archive.New(archiveFile)
	defer a.Close() // nolint: errcheck

	files, err := findFiles(ctx, archive)
	if err != nil {
		return fmt.Errorf("failed to find files to archive: %s", err.Error())
	}
	for _, f := range files {
		log.Debugf("adding %s", f)
		if err = a.Add(wrap(ctx, archive, f, folder), f); err != nil {
			return fmt.Errorf("failed to add %s to the archive: %s", f, err.Error())
		}
	}
	var bins = make([]string, len(binaries))
	for i, binary := range binaries {
		var bin = wrap(ctx, archive, binary.Name, folder)
		log.Debugf("adding %s", bin)
		if err := a.Add(bin, binary.Path); err != nil {
			return fmt.Errorf("failed to add %s -> %s to the archive: %s", binary.Path, binary.Name, err.Error())
		}
		bins[i] = binary.Name
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Type:   artifact.UploadableArchive,
		Name:   folder + "." + format,
		Path:   archivePath,
		Goos:   binaries[0].Goos,
		Goarch: binaries[0].Goarch,
		Goarm:  binaries[0].Goarm,
		Extra: map[string]string{
			"Binaries": strings.Join(bins, ","), // TODO: should have this as a slice
		},
	})
	return nil
}

func skip(ctx *context.Context, archive config.Archive, binaries []artifact.Artifact) error {
	for _, binary := range binaries {
		log.WithField("binary", binary.Name).Info("skip archiving")
		name, err := tmpl.New(ctx).
			WithArtifact(binary, archive.Replacements).
			Apply(archive.NameTemplate)
		if err != nil {
			return err
		}
		binary.Type = artifact.UploadableBinary
		binary.Name = name + binary.Extra["Ext"]
		ctx.Artifacts.Add(binary)
	}
	return nil
}

func findFiles(ctx *context.Context, archive config.Archive) (result []string, err error) {
	for _, glob := range archive.Files {
		files, err := zglob.Glob(glob)
		if err != nil {
			return result, fmt.Errorf("globbing failed for pattern %s: %s", glob, err.Error())
		}
		result = append(result, files...)
	}
	// remove duplicates
	unique.Slice(&result, func(i, j int) bool {
		return strings.Compare(result[i], result[j]) < 0
	})
	return
}

// Wrap archive files with folder if set in config.
func wrap(ctx *context.Context, archive config.Archive, name, folder string) string {
	if archive.WrapInDirectory {
		return filepath.Join(folder, name)
	}
	return name
}

func packageFormat(ctx *context.Context, archive config.Archive, platform string) string {
	for _, override := range archive.FormatOverrides {
		if strings.HasPrefix(platform, override.Goos) {
			return override.Format
		}
	}
	return archive.Format
}
