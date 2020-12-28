// Package nfpm implements the Pipe interface providing nFPM bindings.
package nfpm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/nfpm/v2"
	_ "github.com/goreleaser/nfpm/v2/apk" // blank import to register the format
	_ "github.com/goreleaser/nfpm/v2/deb" // blank import to register the format
	"github.com/goreleaser/nfpm/v2/files"
	_ "github.com/goreleaser/nfpm/v2/rpm" // blank import to register the format
	"github.com/imdario/mergo"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/deprecate"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/linux"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const defaultNameTemplate = "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}{{ if .Mips }}_{{ .Mips }}{{ end }}"

// Pipe for nfpm packaging.
type Pipe struct{}

func (Pipe) String() string {
	return "linux packages"
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	var ids = ids.New("nfpms")
	for i := range ctx.Config.NFPMs {
		var fpm = &ctx.Config.NFPMs[i]
		if fpm.ID == "" {
			fpm.ID = "default"
		}
		if fpm.Bindir == "" {
			fpm.Bindir = "/usr/local/bin"
		}
		if fpm.PackageName == "" {
			fpm.PackageName = ctx.Config.ProjectName
		}
		if fpm.FileNameTemplate == "" {
			fpm.FileNameTemplate = defaultNameTemplate
		}
		if len(fpm.Files) > 0 {
			for src, dst := range fpm.Files {
				fpm.Contents = append(fpm.Contents, &files.Content{
					Source:      src,
					Destination: dst,
				})
			}
			deprecate.Notice(ctx, "nfpms.files")
		}
		if len(fpm.ConfigFiles) > 0 {
			for src, dst := range fpm.ConfigFiles {
				fpm.Contents = append(fpm.Contents, &files.Content{
					Source:      src,
					Destination: dst,
					Type:        "config",
				})
			}
			deprecate.Notice(ctx, "nfpms.config_files")
		}
		if len(fpm.Symlinks) > 0 {
			for src, dst := range fpm.Symlinks {
				fpm.Contents = append(fpm.Contents, &files.Content{
					Source:      src,
					Destination: dst,
					Type:        "symlink",
				})
			}
			deprecate.Notice(ctx, "nfpms.symlinks")
		}
		if len(fpm.RPM.GhostFiles) > 0 {
			for _, dst := range fpm.RPM.GhostFiles {
				fpm.Contents = append(fpm.Contents, &files.Content{
					Destination: dst,
					Type:        "ghost",
					Packager:    "rpm",
				})
			}
			deprecate.Notice(ctx, "nfpms.rpm.ghost_files")
		}
		if len(fpm.RPM.ConfigNoReplaceFiles) > 0 {
			for src, dst := range fpm.RPM.ConfigNoReplaceFiles {
				fpm.Contents = append(fpm.Contents, &files.Content{
					Source:      src,
					Destination: dst,
					Type:        "config|noreplace",
					Packager:    "rpm",
				})
			}
			deprecate.Notice(ctx, "nfpms.rpm.config_noreplace_files")
		}
		if fpm.Deb.VersionMetadata != "" {
			deprecate.Notice(ctx, "nfpms.deb.version_metadata")
			fpm.VersionMetadata = fpm.Deb.VersionMetadata
		}

		if len(fpm.Builds) == 0 {
			for _, b := range ctx.Config.Builds {
				fpm.Builds = append(fpm.Builds, b.ID)
			}
		}
		ids.Inc(fpm.ID)
	}
	return ids.Validate()
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	for _, nfpm := range ctx.Config.NFPMs {
		if len(nfpm.Formats) == 0 {
			// FIXME: this assumes other nfpm configs will fail too...
			return pipe.Skip("no output formats configured")
		}
		if err := doRun(ctx, nfpm); err != nil {
			return err
		}
	}
	return nil
}

func doRun(ctx *context.Context, fpm config.NFPM) error {
	var linuxBinaries = ctx.Artifacts.Filter(artifact.And(
		artifact.ByType(artifact.Binary),
		artifact.ByGoos("linux"),
		artifact.ByIDs(fpm.Builds...),
	)).GroupByPlatform()
	if len(linuxBinaries) == 0 {
		return fmt.Errorf("no linux binaries found for builds %v", fpm.Builds)
	}
	var g = semerrgroup.New(ctx.Parallelism)
	for _, format := range fpm.Formats {
		for platform, artifacts := range linuxBinaries {
			format := format
			arch := linux.Arch(platform)
			artifacts := artifacts
			g.Go(func() error {
				return create(ctx, fpm, format, arch, artifacts)
			})
		}
	}
	return g.Wait()
}

func mergeOverrides(fpm config.NFPM, format string) (*config.NFPMOverridables, error) {
	var overridden config.NFPMOverridables
	if err := mergo.Merge(&overridden, fpm.NFPMOverridables); err != nil {
		return nil, err
	}
	perFormat, ok := fpm.Overrides[format]
	if ok {
		err := mergo.Merge(&overridden, perFormat, mergo.WithOverride)
		if err != nil {
			return nil, err
		}
	}
	return &overridden, nil
}

func create(ctx *context.Context, fpm config.NFPM, format, arch string, binaries []*artifact.Artifact) error {
	overridden, err := mergeOverrides(fpm, format)
	if err != nil {
		return err
	}
	name, err := tmpl.New(ctx).
		WithArtifact(binaries[0], overridden.Replacements).
		WithExtraFields(tmpl.Fields{
			"Release": fpm.Release,
			"Epoch":   fpm.Epoch,
		}).
		Apply(overridden.FileNameTemplate)
	if err != nil {
		return err
	}

	var contents files.Contents
	copy(overridden.Contents, contents)

	// FPM meta package should not contain binaries at all
	if !fpm.Meta {
		var log = log.WithField("package", name+"."+format).WithField("arch", arch)
		for _, binary := range binaries {
			src := binary.Path
			dst := filepath.Join(fpm.Bindir, binary.Name)
			log.WithField("src", src).WithField("dst", dst).Debug("adding binary to package")
			contents = append(contents, &files.Content{
				Source:      src,
				Destination: dst,
			})
		}
	}

	log.WithField("files", contents).Debug("all archive files")

	var info = &nfpm.Info{
		Arch:            arch,
		Platform:        "linux",
		Name:            fpm.PackageName,
		Version:         ctx.Version,
		Section:         "",
		Priority:        "",
		Epoch:           fpm.Epoch,
		Release:         fpm.Release,
		Prerelease:      fpm.Prerelease,
		VersionMetadata: fpm.VersionMetadata,
		Maintainer:      fpm.Maintainer,
		Description:     fpm.Description,
		Vendor:          fpm.Vendor,
		Homepage:        fpm.Homepage,
		License:         fpm.License,
		Overridables: nfpm.Overridables{
			Conflicts:    overridden.Conflicts,
			Depends:      overridden.Dependencies,
			Recommends:   overridden.Recommends,
			Suggests:     overridden.Suggests,
			Replaces:     overridden.Replaces,
			EmptyFolders: overridden.EmptyFolders,
			Contents:     contents,
			Scripts: nfpm.Scripts{
				PreInstall:  overridden.Scripts.PreInstall,
				PostInstall: overridden.Scripts.PostInstall,
				PreRemove:   overridden.Scripts.PreRemove,
				PostRemove:  overridden.Scripts.PostRemove,
			},
			Deb: nfpm.Deb{
				Scripts: nfpm.DebScripts{
					Rules:     overridden.Deb.Scripts.Rules,
					Templates: overridden.Deb.Scripts.Templates,
				},
				Triggers: nfpm.DebTriggers{
					Interest:        overridden.Deb.Triggers.Interest,
					InterestAwait:   overridden.Deb.Triggers.InterestAwait,
					InterestNoAwait: overridden.Deb.Triggers.InterestNoAwait,
					Activate:        overridden.Deb.Triggers.Activate,
					ActivateAwait:   overridden.Deb.Triggers.ActivateAwait,
					ActivateNoAwait: overridden.Deb.Triggers.ActivateNoAwait,
				},
				Breaks: overridden.Deb.Breaks,
				Signature: nfpm.DebSignature{
					KeyFile:       overridden.Deb.Signature.KeyFile,
					KeyPassphrase: getPassphraseFromEnv(ctx, "DEB", fpm.ID),
					Type:          overridden.Deb.Signature.Type,
				},
			},
			RPM: nfpm.RPM{
				Summary:     overridden.RPM.Summary,
				Group:       overridden.RPM.Group,
				Compression: overridden.RPM.Compression,
				Signature: nfpm.RPMSignature{
					KeyFile:       overridden.RPM.Signature.KeyFile,
					KeyPassphrase: getPassphraseFromEnv(ctx, "RPM", fpm.ID),
				},
			},
			APK: nfpm.APK{
				Signature: nfpm.APKSignature{
					KeyFile:       overridden.APK.Signature.KeyFile,
					KeyPassphrase: getPassphraseFromEnv(ctx, "APK", fpm.ID),
					KeyName:       overridden.APK.Signature.KeyName,
				},
			},
		},
	}

	if ctx.SkipSign {
		info.APK.Signature = nfpm.APKSignature{}
		info.RPM.Signature = nfpm.RPMSignature{}
		info.Deb.Signature = nfpm.DebSignature{}
	}

	if err = nfpm.Validate(info); err != nil {
		return fmt.Errorf("invalid nfpm config: %w", err)
	}

	packager, err := nfpm.Get(format)
	if err != nil {
		return err
	}

	var path = filepath.Join(ctx.Config.Dist, name+"."+format)
	log.WithField("file", path).Info("creating")
	w, err := os.Create(path)
	if err != nil {
		return err
	}
	defer w.Close()
	if err := packager.Package(nfpm.WithDefaults(info), w); err != nil {
		return fmt.Errorf("nfpm failed: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("could not close package file: %w", err)
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Type:   artifact.LinuxPackage,
		Name:   name + "." + format,
		Path:   path,
		Goos:   binaries[0].Goos,
		Goarch: binaries[0].Goarch,
		Goarm:  binaries[0].Goarm,
		Extra: map[string]interface{}{
			"Builds": binaries,
			"ID":     fpm.ID,
			"Format": format,
			"Files":  contents,
		},
	})
	return nil
}

func getPassphraseFromEnv(ctx *context.Context, packager string, nfpmID string) string {
	var passphrase string

	nfpmID = strings.ToUpper(nfpmID)
	packagerSpecificPassphrase := ctx.Env[fmt.Sprintf(
		"NFPM_%s_%s_PASSPHRASE",
		nfpmID,
		packager,
	)]
	if packagerSpecificPassphrase != "" {
		passphrase = packagerSpecificPassphrase
	} else {
		generalPassphrase := ctx.Env[fmt.Sprintf("NFPM_%s_PASSPHRASE", nfpmID)]
		passphrase = generalPassphrase
	}

	return passphrase
}
