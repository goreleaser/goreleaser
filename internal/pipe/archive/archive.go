// Package archive implements the pipe interface with the intent of
// archiving and compressing the binaries, readme, and other artifacts. It
// also provides an Archive interface which represents an archiving format.
package archive

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/archivefiles"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/deprecate"
	"github.com/goreleaser/goreleaser/v2/internal/ids"
	"github.com/goreleaser/goreleaser/v2/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/archive"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const (
	defaultNameTemplateSuffix = `{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ with .Arm }}v{{ . }}{{ end }}{{ with .Mips }}_{{ . }}{{ end }}{{ if not (eq .Amd64 "v1") }}{{ .Amd64 }}{{ end }}`
	defaultNameTemplate       = "{{ .ProjectName }}_" + defaultNameTemplateSuffix
	defaultBinaryNameTemplate = "{{ .Binary }}_" + defaultNameTemplateSuffix
)

// ErrArchiveDifferentBinaryCount happens when an archive uses several builds which have different goos/goarch/etc sets,
// causing the archives for some platforms to have more binaries than others.
// GoReleaser breaks in these cases as it will only cause confusion to other users.
var ErrArchiveDifferentBinaryCount = errors.New("archive has different count of binaries for each platform, which may cause your users confusion.\nLearn more at https://goreleaser.com/errors/multiple-binaries-archive")

//nolint:gochecknoglobals
var lock sync.Mutex

// Pipe for archive.
type Pipe struct{}

func (Pipe) String() string {
	return "archives"
}

func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Archive)
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("archives")
	if len(ctx.Config.Archives) == 0 {
		ctx.Config.Archives = append(ctx.Config.Archives, config.Archive{})
	}
	for i := range ctx.Config.Archives {
		archive := &ctx.Config.Archives[i]
		if archive.Format != "" {
			deprecate.Notice(ctx, "archives.format")
			archive.Formats = append(archive.Formats, archive.Format)
		}
		if len(archive.Formats) == 0 {
			archive.Formats = []string{"tar.gz"}
		}
		for i := range archive.FormatOverrides {
			over := &archive.FormatOverrides[i]
			if over.Format != "" {
				deprecate.Notice(ctx, "archives.format_overrides.format")
				over.Formats = append(over.Formats, over.Format)
			}
		}
		if archive.ID == "" {
			archive.ID = "default"
		}
		if len(archive.Builds) > 0 {
			deprecate.Notice(ctx, "archives.builds")
			archive.IDs = append(archive.IDs, archive.Builds...)
		}
		if len(archive.Files) == 0 {
			archive.Files = []config.File{
				{Source: "license*", Default: true},
				{Source: "LICENSE*", Default: true},
				{Source: "readme*", Default: true},
				{Source: "README*", Default: true},
				{Source: "changelog*", Default: true},
				{Source: "CHANGELOG*", Default: true},
			}
		}
		if archive.NameTemplate == "" {
			archive.NameTemplate = defaultNameTemplate
			if slices.Contains(archive.Formats, "binary") {
				archive.NameTemplate = defaultBinaryNameTemplate
			}
		}
		archive.BuildsInfo.Mode = 0o755
		ids.Inc(archive.ID)
	}
	return ids.Validate()
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	g := semerrgroup.New(ctx.Parallelism)
	for i, archive := range ctx.Config.Archives {
		if archive.Meta {
			for _, format := range archive.Formats {
				g.Go(func() error {
					return createMeta(ctx, archive, format)
				})
			}
			continue
		}

		filter := []artifact.Filter{artifact.Or(
			artifact.ByType(artifact.Binary),
			artifact.ByType(artifact.UniversalBinary),
			artifact.ByType(artifact.Header),
			artifact.ByType(artifact.CArchive),
			artifact.ByType(artifact.CShared),
		)}
		if len(archive.IDs) > 0 {
			filter = append(filter, artifact.ByIDs(archive.IDs...))
		}

		isBinary := slices.Contains(archive.Formats, "binary")
		artifacts := ctx.Artifacts.Filter(artifact.And(filter...)).GroupByPlatform()
		if err := checkArtifacts(artifacts); err != nil && !isBinary && !archive.AllowDifferentBinaryCount {
			return fmt.Errorf("invalid archive: %d: %w", i, ErrArchiveDifferentBinaryCount)
		}
		for group, artifacts := range artifacts {
			log.Debugf("group %s has %d binaries", group, len(artifacts))
			formats := packageFormats(archive, artifacts[0].Goos)
			for _, format := range formats {
				switch format {
				case "none":
					// do nothing
					log.WithField("goos", artifacts[0].Goos).Info("ignored due to format override to 'none'")
				case "binary":
					g.Go(func() error {
						return skip(ctx, archive, artifacts)
					})
				default:
					g.Go(func() error {
						return create(ctx, archive, artifacts, format)
					})
				}
			}
		}
	}
	return g.Wait()
}

func checkArtifacts(artifacts map[string][]*artifact.Artifact) error {
	lens := map[int]bool{}
	for _, v := range artifacts {
		lens[len(v)] = true
	}
	if len(lens) <= 1 {
		return nil
	}
	return ErrArchiveDifferentBinaryCount
}

func createMeta(ctx *context.Context, arch config.Archive, format string) error {
	return create(ctx, arch, nil, format)
}

func create(ctx *context.Context, arch config.Archive, binaries []*artifact.Artifact, format string) error {
	template := tmpl.New(ctx)
	if len(binaries) > 0 {
		template = template.WithArtifact(binaries[0])
	}
	folder, err := template.Apply(arch.NameTemplate)
	if err != nil {
		return err
	}
	var archivePath string
	if format == "makeself" {
		// For makeself archives, use custom extension or default to .run
		extension := arch.Makeself.Extension
		if extension == "" {
			extension = ".run"
		}
		// Ensure extension starts with a dot
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		// Apply template to extension
		extension, err = template.Apply(extension)
		if err != nil {
			return fmt.Errorf("failed to apply template to makeself extension: %w", err)
		}
		archivePath = filepath.Join(ctx.Config.Dist, folder+extension)
	} else {
		archivePath = filepath.Join(ctx.Config.Dist, folder+"."+format)
	}
	lock.Lock()
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755|os.ModeDir); err != nil {
		lock.Unlock()
		return err
	}
	if _, err = os.Stat(archivePath); !errors.Is(err, fs.ErrNotExist) {
		lock.Unlock()
		return fmt.Errorf("archive named %s already exists. Check your archive name template", archivePath)
	}
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		lock.Unlock()
		return fmt.Errorf("failed to create directory %s: %w", archivePath, err)
	}
	lock.Unlock()
	defer archiveFile.Close()

	log := log.WithField("name", archivePath)
	log.Info("archiving")

	wrap, err := template.Apply(wrapFolder(arch))
	if err != nil {
		return err
	}
	var a archive.Archive
	if format == "makeself" {
		// Create makeself archive with configuration
		makeselfConfig := arch.Makeself
		// Apply templates to configuration fields
		if makeselfConfig.Label != "" {
			makeselfConfig.Label, err = template.Apply(makeselfConfig.Label)
			if err != nil {
				return fmt.Errorf("failed to apply template to makeself label: %w", err)
			}
		}
		if makeselfConfig.InstallScript != "" {
			makeselfConfig.InstallScript, err = template.Apply(makeselfConfig.InstallScript)
			if err != nil {
				return fmt.Errorf("failed to apply template to makeself install_script: %w", err)
			}
		}
		if makeselfConfig.InstallScriptFile != "" {
			makeselfConfig.InstallScriptFile, err = template.Apply(makeselfConfig.InstallScriptFile)
			if err != nil {
				return fmt.Errorf("failed to apply template to makeself install_script_file: %w", err)
			}
		}
		if makeselfConfig.Compression != "" {
			makeselfConfig.Compression, err = template.Apply(makeselfConfig.Compression)
			if err != nil {
				return fmt.Errorf("failed to apply template to makeself compression: %w", err)
			}
		}
		// Apply templates to LSM configuration
		if makeselfConfig.LSMTemplate != "" {
			makeselfConfig.LSMTemplate, err = template.Apply(makeselfConfig.LSMTemplate)
			if err != nil {
				return fmt.Errorf("failed to apply template to makeself lsm_template: %w", err)
			}
		}
		if makeselfConfig.LSMFile != "" {
			makeselfConfig.LSMFile, err = template.Apply(makeselfConfig.LSMFile)
			if err != nil {
				return fmt.Errorf("failed to apply template to makeself lsm_file: %w", err)
			}
		}
		// Apply templates to extension
		if makeselfConfig.Extension != "" {
			makeselfConfig.Extension, err = template.Apply(makeselfConfig.Extension)
			if err != nil {
				return fmt.Errorf("failed to apply template to makeself extension: %w", err)
			}
		}
		// Apply templates to extra args
		for i, arg := range makeselfConfig.ExtraArgs {
			makeselfConfig.ExtraArgs[i], err = template.Apply(arg)
			if err != nil {
				return fmt.Errorf("failed to apply template to makeself extra_args[%d]: %w", i, err)
			}
		}
		a, err = archive.NewWithMakeselfConfig(archiveFile, archivePath, makeselfConfig)
		if err != nil {
			return err
		}
	} else {
		a, err = archive.New(archiveFile, format)
		if err != nil {
			return err
		}
	}
	a = NewEnhancedArchive(a, wrap)
	defer a.Close()

	files, err := archivefiles.Eval(template, arch.Files)
	if err != nil {
		return fmt.Errorf("failed to find files to archive: %w", err)
	}
	if arch.Meta && len(files) == 0 {
		return errors.New("no files found")
	}
	for _, f := range files {
		if err = a.Add(f); err != nil {
			return fmt.Errorf("failed to add: '%s' -> '%s': %w", f.Source, f.Destination, err)
		}
	}
	bins := []string{}
	for _, binary := range binaries {
		dst := binary.Name
		if arch.StripBinaryDirectory {
			dst = filepath.Base(dst)
		}
		if err := a.Add(config.File{
			Source:      binary.Path,
			Destination: dst,
			Info:        arch.BuildsInfo,
		}); err != nil {
			return fmt.Errorf("failed to add: '%s' -> '%s': %w", binary.Path, dst, err)
		}
		bins = append(bins, binary.Name)
	}
	// Determine artifact name based on format and extension
	var artifactName string
	if format == "makeself" {
		// For makeself archives, use custom extension or default to .run
		extension := arch.Makeself.Extension
		if extension == "" {
			extension = ".run"
		}
		// Ensure extension starts with a dot
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		// Apply template to extension
		extension, err = template.Apply(extension)
		if err != nil {
			return fmt.Errorf("failed to apply template to makeself extension for artifact name: %w", err)
		}
		artifactName = folder + extension
	} else {
		artifactName = folder + "." + format
	}

	art := &artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: artifactName,
		Path: archivePath,
		Extra: map[string]any{
			artifact.ExtraID:        arch.ID,
			artifact.ExtraFormat:    format,
			artifact.ExtraWrappedIn: wrap,
			artifact.ExtraBinaries:  bins,
		},
	}
	if len(binaries) > 0 {
		art.Goos = binaries[0].Goos
		art.Goarch = binaries[0].Goarch
		art.Goamd64 = binaries[0].Goamd64
		art.Go386 = binaries[0].Go386
		art.Goarm = binaries[0].Goarm
		art.Goarm64 = binaries[0].Goarm64
		art.Gomips = binaries[0].Gomips
		art.Goppc64 = binaries[0].Goppc64
		art.Goriscv64 = binaries[0].Goriscv64
		art.Target = binaries[0].Target
		if rep, ok := binaries[0].Extra[artifact.ExtraReplaces]; ok {
			art.Extra[artifact.ExtraReplaces] = rep
		}
	}

	ctx.Artifacts.Add(art)
	return nil
}

func wrapFolder(a config.Archive) string {
	switch a.WrapInDirectory {
	case "true":
		return a.NameTemplate
	case "false":
		return ""
	default:
		return a.WrapInDirectory
	}
}

func skip(ctx *context.Context, archive config.Archive, binaries []*artifact.Artifact) error {
	for _, binary := range binaries {
		name, err := tmpl.New(ctx).WithArtifact(binary).Apply(archive.NameTemplate)
		if err != nil {
			return err
		}
		finalName := name + binary.Ext()
		log.WithField("binary", binary.Name).
			WithField("name", finalName).
			Info("archiving")

		art := &artifact.Artifact{
			Type:      artifact.UploadableBinary,
			Name:      finalName,
			Path:      binary.Path,
			Goos:      binary.Goos,
			Goarch:    binary.Goarch,
			Goamd64:   binary.Goamd64,
			Go386:     binary.Go386,
			Goarm:     binary.Goarm,
			Goarm64:   binary.Goarm64,
			Gomips:    binary.Gomips,
			Goppc64:   binary.Goppc64,
			Goriscv64: binary.Goriscv64,
			Target:    binary.Target,
			Extra: map[string]any{
				artifact.ExtraID:     archive.ID,
				artifact.ExtraFormat: "binary",
				artifact.ExtraBinary: binary.Name,
			},
		}
		if rep, ok := binaries[0].Extra[artifact.ExtraReplaces]; ok {
			art.Extra[artifact.ExtraReplaces] = rep
		}
		ctx.Artifacts.Add(art)
	}
	return nil
}

func packageFormats(archive config.Archive, platform string) []string {
	for _, override := range archive.FormatOverrides {
		if override.Goos == "" {
			log.Warn("override has no goos, ignoring")
			continue
		}
		if len(override.Formats) == 0 {
			log.WithField("goos", override.Goos).
				Warn("override has no formats, ignoring")
			continue
		}
		if strings.HasPrefix(platform, override.Goos) {
			return override.Formats
		}
	}
	return archive.Formats
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

// Add adds a file.
func (d EnhancedArchive) Add(f config.File) error {
	name := strings.ReplaceAll(filepath.Join(d.wrap, f.Destination), "\\", "/")
	log.Debugf("adding file: %s as %s", f.Source, name)
	ff := config.File{
		Source:      f.Source,
		Destination: name,
		Info:        f.Info,
	}
	return d.a.Add(ff)
}

// Close closes the underlying archive.
func (d EnhancedArchive) Close() error {
	return d.a.Close()
}
