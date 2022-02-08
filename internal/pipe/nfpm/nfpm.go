// Package nfpm implements the Pipe interface providing nFPM bindings.
package nfpm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/deprecate"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/goreleaser/nfpm/v2"
	"github.com/goreleaser/nfpm/v2/deprecation"
	"github.com/goreleaser/nfpm/v2/files"
	"github.com/imdario/mergo"

	_ "github.com/goreleaser/nfpm/v2/apk" // blank import to register the format
	_ "github.com/goreleaser/nfpm/v2/deb" // blank import to register the format
	_ "github.com/goreleaser/nfpm/v2/rpm" // blank import to register the format
)

const (
	defaultNameTemplate = "{{ .PackageName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}{{ if .Mips }}_{{ .Mips }}{{ end }}"
	extraFiles          = "Files"
)

// Pipe for nfpm packaging.
type Pipe struct{}

func (Pipe) String() string                 { return "linux packages" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.NFPMs) == 0 }

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("nfpms")
	for i := range ctx.Config.NFPMs {
		fpm := &ctx.Config.NFPMs[i]
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
		if len(fpm.Builds) == 0 { // TODO: change this to empty by default and deal with it in the filtering code
			for _, b := range ctx.Config.Builds {
				fpm.Builds = append(fpm.Builds, b.ID)
			}
		}
		ids.Inc(fpm.ID)
	}

	deprecation.Noticer = deprecate.NewWriter(ctx)
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
	linuxBinaries := ctx.Artifacts.Filter(artifact.And(
		artifact.ByType(artifact.Binary),
		artifact.ByGoos("linux"),
		artifact.ByIDs(fpm.Builds...),
	)).GroupByPlatform()
	if len(linuxBinaries) == 0 {
		return fmt.Errorf("no linux binaries found for builds %v", fpm.Builds)
	}
	g := semerrgroup.New(ctx.Parallelism)
	for _, format := range fpm.Formats {
		for _, artifacts := range linuxBinaries {
			format := format
			artifacts := artifacts
			g.Go(func() error {
				return create(ctx, fpm, format, artifacts)
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

func create(ctx *context.Context, fpm config.NFPM, format string, binaries []*artifact.Artifact) error {
	arch := binaries[0].Goarch + binaries[0].Goarm + binaries[0].Gomips

	overridden, err := mergeOverrides(fpm, format)
	if err != nil {
		return err
	}
	t := tmpl.New(ctx).
		WithArtifact(binaries[0], overridden.Replacements).
		WithExtraFields(tmpl.Fields{
			"Release":     fpm.Release,
			"Epoch":       fpm.Epoch,
			"PackageName": fpm.PackageName,
		})

	binDir, err := t.Apply(fpm.Bindir)
	if err != nil {
		return err
	}

	homepage, err := t.Apply(fpm.Homepage)
	if err != nil {
		return err
	}

	description, err := t.Apply(fpm.Description)
	if err != nil {
		return err
	}

	maintainer, err := t.Apply(fpm.Maintainer)
	if err != nil {
		return err
	}

	debKeyFile, err := t.Apply(overridden.Deb.Signature.KeyFile)
	if err != nil {
		return err
	}

	rpmKeyFile, err := t.Apply(overridden.RPM.Signature.KeyFile)
	if err != nil {
		return err
	}

	apkKeyFile, err := t.Apply(overridden.APK.Signature.KeyFile)
	if err != nil {
		return err
	}

	contents := files.Contents{}
	for _, content := range overridden.Contents {
		src, err := t.Apply(content.Source)
		if err != nil {
			return err
		}
		dst, err := t.Apply(content.Destination)
		if err != nil {
			return err
		}
		contents = append(contents, &files.Content{
			Source:      src,
			Destination: dst,
			Type:        content.Type,
			Packager:    content.Packager,
			FileInfo:    content.FileInfo,
		})
	}

	if len(fpm.Deb.Lintian) > 0 {
		lines := make([]string, 0, len(fpm.Deb.Lintian))
		for _, ov := range fpm.Deb.Lintian {
			lines = append(lines, fmt.Sprintf("%s: %s", fpm.PackageName, ov))
		}
		lintianPath := filepath.Join(ctx.Config.Dist, "deb", fpm.PackageName, ".lintian")
		if err := os.MkdirAll(filepath.Dir(lintianPath), 0o755); err != nil {
			return fmt.Errorf("failed to write lintian file: %w", err)
		}
		if err := os.WriteFile(lintianPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
			return fmt.Errorf("failed to write lintian file: %w", err)
		}

		log.Infof("creating %q", lintianPath)
		contents = append(contents, &files.Content{
			Source:      lintianPath,
			Destination: filepath.Join("./usr/share/lintian/overrides", fpm.PackageName),
			Packager:    "deb",
			FileInfo: &files.ContentFileInfo{
				Mode: 0o644,
			},
		})
	}

	log := log.WithField("package", fpm.PackageName).WithField("format", format).WithField("arch", arch)

	// FPM meta package should not contain binaries at all
	if !fpm.Meta {
		for _, binary := range binaries {
			src := binary.Path
			dst := filepath.Join(binDir, binary.Name)
			log.WithField("src", src).WithField("dst", dst).Debug("adding binary to package")
			contents = append(contents, &files.Content{
				Source:      filepath.ToSlash(src),
				Destination: filepath.ToSlash(dst),
				FileInfo: &files.ContentFileInfo{
					Mode: 0o755,
				},
			})
		}
	}

	log.WithField("files", destinations(contents)).Debug("all archive files")

	info := &nfpm.Info{
		Arch:            arch,
		Platform:        "linux",
		Name:            fpm.PackageName,
		Version:         ctx.Version,
		Section:         fpm.Section,
		Priority:        fpm.Priority,
		Epoch:           fpm.Epoch,
		Release:         fpm.Release,
		Prerelease:      fpm.Prerelease,
		VersionMetadata: fpm.VersionMetadata,
		Maintainer:      maintainer,
		Description:     description,
		Vendor:          fpm.Vendor,
		Homepage:        homepage,
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
					PackageSignature: nfpm.PackageSignature{
						KeyFile:       debKeyFile,
						KeyPassphrase: getPassphraseFromEnv(ctx, "DEB", fpm.ID),
					},
					Type: overridden.Deb.Signature.Type,
				},
			},
			RPM: nfpm.RPM{
				Summary:     overridden.RPM.Summary,
				Group:       overridden.RPM.Group,
				Compression: overridden.RPM.Compression,
				Signature: nfpm.RPMSignature{
					PackageSignature: nfpm.PackageSignature{
						KeyFile:       rpmKeyFile,
						KeyPassphrase: getPassphraseFromEnv(ctx, "RPM", fpm.ID),
					},
				},
				Scripts: nfpm.RPMScripts{
					PreTrans:  overridden.RPM.Scripts.PreTrans,
					PostTrans: overridden.RPM.Scripts.PostTrans,
				},
			},
			APK: nfpm.APK{
				Signature: nfpm.APKSignature{
					PackageSignature: nfpm.PackageSignature{
						KeyFile:       apkKeyFile,
						KeyPassphrase: getPassphraseFromEnv(ctx, "APK", fpm.ID),
					},
					KeyName: overridden.APK.Signature.KeyName,
				},
				Scripts: nfpm.APKScripts{
					PreUpgrade:  overridden.APK.Scripts.PreUpgrade,
					PostUpgrade: overridden.APK.Scripts.PostUpgrade,
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

	info = nfpm.WithDefaults(info)
	name, err := t.WithExtraFields(tmpl.Fields{
		"ConventionalFileName": packager.ConventionalFileName(info),
	}).Apply(overridden.FileNameTemplate)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(name, "."+format) {
		name = name + "." + format
	}

	path := filepath.Join(ctx.Config.Dist, name)
	log.WithField("file", path).Info("creating")
	w, err := os.Create(path)
	if err != nil {
		return err
	}
	defer w.Close()
	if err := packager.Package(info, w); err != nil {
		return fmt.Errorf("nfpm failed: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("could not close package file: %w", err)
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Type:   artifact.LinuxPackage,
		Name:   name,
		Path:   path,
		Goos:   binaries[0].Goos,
		Goarch: binaries[0].Goarch,
		Goarm:  binaries[0].Goarm,
		Extra: map[string]interface{}{
			artifact.ExtraBuilds: binaries,
			artifact.ExtraID:     fpm.ID,
			artifact.ExtraFormat: format,
			extraFiles:           contents,
		},
	})
	return nil
}

func destinations(contents files.Contents) []string {
	result := make([]string, 0, len(contents))
	for _, f := range contents {
		result = append(result, f.Destination)
	}
	return result
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
