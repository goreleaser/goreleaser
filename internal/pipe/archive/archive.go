// Package archive implements the pipe interface with the intent of
// archiving and compressing the binaries, readme, and other artifacts. It
// also provides an Archive interface which represents an archiving format.
package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/apex/log"
	"github.com/campoy/unique"
	zglob "github.com/mattn/go-zglob"
	"golang.org/x/sync/errgroup"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/archive"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	defaultNameTemplate       = "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"
	defaultBinaryNameTemplate = "{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"
)

// nolint: gochecknoglobals
var lock sync.Mutex

// Pipe for archive
type Pipe struct{}

func (Pipe) String() string {
	return "archives"
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	var archive = &ctx.Config.Archive
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
	return nil
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	var g errgroup.Group // TODO: use semerrgroup here
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

func create(ctx *context.Context, binaries []artifact.Artifact) error {
	var format = packageFormat(ctx, binaries[0].Goos)
	folder, err := tmpl.New(ctx).
		WithArtifact(binaries[0], ctx.Config.Archive.Replacements).
		Apply(ctx.Config.Archive.NameTemplate)
	if err != nil {
		return err
	}
	archivePath := filepath.Join(ctx.Config.Dist, folder+"."+format)
	lock.Lock()
	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		lock.Unlock()
		return fmt.Errorf("archive named %s already exists. Check your archive name template", archivePath)
	}
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		lock.Unlock()
		return fmt.Errorf("failed to create directory %s: %s", archivePath, err.Error())
	}
	lock.Unlock()
	defer archiveFile.Close() // nolint: errcheck
	var log = log.WithField("archive", archivePath)
	log.Info("creating")
	var wrap string
	if ctx.Config.Archive.WrapInDirectory {
		wrap = folder
	}
	var a = NewEnhancedArchive(archive.New(archiveFile), wrap)
	defer a.Close() // nolint: errcheck

	files, err := findFiles(ctx)
	if err != nil {
		return fmt.Errorf("failed to find files to archive: %s", err.Error())
	}
	for _, f := range files {
		if err = a.Add(f, f); err != nil {
			return fmt.Errorf("failed to add %s to the archive: %s", f, err.Error())
		}
	}
	for _, binary := range binaries {
		if err := a.Add(binary.Name, binary.Path); err != nil {
			return fmt.Errorf("failed to add %s -> %s to the archive: %s", binary.Path, binary.Name, err.Error())
		}
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Type:   artifact.UploadableArchive,
		Name:   folder + "." + format,
		Path:   archivePath,
		Goos:   binaries[0].Goos,
		Goarch: binaries[0].Goarch,
		Goarm:  binaries[0].Goarm,
	})
	return nil
}

func skip(ctx *context.Context, binaries []artifact.Artifact) error {
	for _, binary := range binaries {
		log.WithField("binary", binary.Name).Info("skip archiving")
		name, err := tmpl.New(ctx).
			WithArtifact(binary, ctx.Config.Archive.Replacements).
			Apply(ctx.Config.Archive.NameTemplate)
		if err != nil {
			return err
		}
		binary.Type = artifact.UploadableBinary
		binary.Name = name + binary.Extra["Ext"]
		ctx.Artifacts.Add(binary)
	}
	return nil
}

func findFiles(ctx *context.Context) (result []string, err error) {
	for _, glob := range ctx.Config.Archive.Files {
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

func packageFormat(ctx *context.Context, platform string) string {
	for _, override := range ctx.Config.Archive.FormatOverrides {
		if strings.HasPrefix(platform, override.Goos) {
			return override.Format
		}
	}
	return ctx.Config.Archive.Format
}

// NewEnhancedArchive enhances a pre-existing archive.Archive instance
// with this pipe specifics.
func NewEnhancedArchive(a archive.Archive, wrap string) archive.Archive {
	return EnhancedArchive{
		a:     a,
		wrap:  wrap,
		files: map[string]string{},
	}
}

// EnhancedArchive is an archive.Archive implementation which decorates an
// archive.Archive adding wrap directory support, logging and windows
// backslash fixes.
type EnhancedArchive struct {
	a     archive.Archive
	wrap  string
	files map[string]string
}

// Add adds a file
func (d EnhancedArchive) Add(name, path string) error {
	name = strings.Replace(filepath.Join(d.wrap, name), "\\", "/", -1)
	log.Debugf("adding file: %s as %s", path, name)
	if _, ok := d.files[name]; ok {
		return fmt.Errorf("file %s already exists in the archive", name)
	}
	d.files[name] = path
	return d.a.Add(name, path)
}

// Close closes the underlying archive
func (d EnhancedArchive) Close() error {
	return d.a.Close()
}
