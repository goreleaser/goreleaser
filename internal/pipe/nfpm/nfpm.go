// Package nfpm implements the Pipe interface providing nFPM bindings.
package nfpm

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"dario.cat/mergo"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/deprecate"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/goreleaser/nfpm/v2"
	"github.com/goreleaser/nfpm/v2/deprecation"
	"github.com/goreleaser/nfpm/v2/files"

	_ "github.com/goreleaser/nfpm/v2/apk"  // blank import to register the format
	_ "github.com/goreleaser/nfpm/v2/arch" // blank import to register the format
	_ "github.com/goreleaser/nfpm/v2/deb"  // blank import to register the format
	_ "github.com/goreleaser/nfpm/v2/rpm"  // blank import to register the format
)

const (
	defaultNameTemplate = `{{ .PackageName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ with .Arm }}v{{ . }}{{ end }}{{ with .Mips }}_{{ . }}{{ end }}{{ if not (eq .Amd64 "v1") }}{{ .Amd64 }}{{ end }}`
	extraFiles          = "Files"
)

// Pipe for nfpm packaging.
type Pipe struct{}

func (Pipe) String() string { return "linux packages" }
func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.NFPM) || len(ctx.Config.NFPMs) == 0
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("nfpms")
	for i := range ctx.Config.NFPMs {
		fpm := &ctx.Config.NFPMs[i]
		if fpm.ID == "" {
			fpm.ID = "default"
		}
		if fpm.Bindir == "" {
			fpm.Bindir = "/usr/bin"
		}
		if fpm.Libdirs.Header == "" {
			fpm.Libdirs.Header = "/usr/include"
		}
		if fpm.Libdirs.CShared == "" {
			fpm.Libdirs.CShared = "/usr/lib"
		}
		if fpm.Libdirs.CArchive == "" {
			fpm.Libdirs.CArchive = "/usr/lib"
		}
		if fpm.PackageName == "" {
			fpm.PackageName = ctx.Config.ProjectName
		}
		if fpm.FileNameTemplate == "" {
			fpm.FileNameTemplate = defaultNameTemplate
		}
		if fpm.Maintainer == "" {
			deprecate.NoticeCustom(ctx, "nfpms.maintainer", "`{{ .Property }}` should always be set, check {{ .URL }} for more info")
		}
		ids.Inc(fpm.ID)
	}

	deprecation.Noticer = io.Discard
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
	filters := []artifact.Filter{
		artifact.Or(
			artifact.ByType(artifact.Binary),
			artifact.ByType(artifact.Header),
			artifact.ByType(artifact.CArchive),
			artifact.ByType(artifact.CShared),
		),
		artifact.Or(
			artifact.ByGoos("linux"),
			artifact.ByGoos("ios"),
			artifact.ByGoos("android"),
		),
	}
	if len(fpm.Builds) > 0 {
		filters = append(filters, artifact.ByIDs(fpm.Builds...))
	}
	linuxBinaries := ctx.Artifacts.
		Filter(artifact.And(filters...)).
		GroupByPlatform()
	if len(linuxBinaries) == 0 {
		return fmt.Errorf("no linux binaries found for builds %v", fpm.Builds)
	}
	g := semerrgroup.New(ctx.Parallelism)
	for _, format := range fpm.Formats {
		for _, artifacts := range linuxBinaries {
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

const termuxFormat = "termux.deb"

func isSupportedTermuxArch(goos, goarch string) bool {
	if goos != "android" {
		return false
	}
	for _, arch := range []string{"amd64", "arm64", "386"} {
		if strings.HasPrefix(goarch, arch) {
			return true
		}
	}
	return false
}

// arch officially only supports x86_64.
// however, there are unofficial ports for 686, arm64, and armv7
func isSupportedArchlinuxArch(goarch, goarm string) bool {
	if goarch == "arm" && goarm == "7" {
		return true
	}
	for _, arch := range []string{"amd64", "arm64", "386"} {
		if strings.HasPrefix(goarch, arch) {
			return true
		}
	}
	return false
}

var termuxArchReplacer = strings.NewReplacer(
	"386", "i686",
	"amd64", "x86_64",
	"arm64", "aarch64",
)

func create(ctx *context.Context, fpm config.NFPM, format string, artifacts []*artifact.Artifact) error {
	// TODO: improve mips handling on nfpm
	infoArch := artifacts[0].Goarch + artifacts[0].Goarm + artifacts[0].Gomips // key used for the ConventionalFileName et al
	arch := infoArch + artifacts[0].Goamd64                                    // unique arch key
	infoPlatform := artifacts[0].Goos
	if infoPlatform == "ios" {
		if format == "deb" {
			infoPlatform = "iphoneos-arm64"
		} else {
			log.Debugf("skipping ios for %s as its not supported", format)
			return nil
		}
	}

	switch format {
	case "archlinux":
		if !isSupportedArchlinuxArch(artifacts[0].Goarch, artifacts[0].Goarm) {
			log.Debugf("skipping archlinux for %s as its not supported", arch)
			return nil
		}
	case termuxFormat:
		if !isSupportedTermuxArch(artifacts[0].Goos, artifacts[0].Goarch) {
			log.Debugf("skipping termux.deb for %s as its not supported by termux", arch)
			return nil
		}

		infoArch = termuxArchReplacer.Replace(infoArch)
		arch = termuxArchReplacer.Replace(arch)
		infoPlatform = "linux"
		fpm.Bindir = termuxPrefixedDir(fpm.Bindir)
		fpm.Libdirs.Header = termuxPrefixedDir(fpm.Libdirs.Header)
		fpm.Libdirs.CArchive = termuxPrefixedDir(fpm.Libdirs.CArchive)
		fpm.Libdirs.CShared = termuxPrefixedDir(fpm.Libdirs.CShared)
	}

	if artifacts[0].Goos == "android" && format != termuxFormat {
		log.Debugf("skipping android packaging as its not supported by %s", format)
		return nil
	}

	overridden, err := mergeOverrides(fpm, format)
	if err != nil {
		return err
	}

	packageName, err := tmpl.New(ctx).Apply(fpm.PackageName)
	if err != nil {
		return err
	}

	t := tmpl.New(ctx).
		WithArtifact(artifacts[0]).
		WithExtraFields(tmpl.Fields{
			"Release":     fpm.Release,
			"Epoch":       fpm.Epoch,
			"PackageName": packageName,
		})

	if err := t.ApplyAll(
		&fpm.Bindir,
		&fpm.Homepage,
		&fpm.Description,
		&fpm.Maintainer,
		&overridden.Scripts.PostInstall,
		&overridden.Scripts.PreInstall,
		&overridden.Scripts.PostRemove,
		&overridden.Scripts.PreRemove,
	); err != nil {
		return err
	}

	// We cannot use t.ApplyAll on the following fields as they are shared
	// across multiple nfpms.
	//
	t = t.WithExtraFields(tmpl.Fields{
		"Format": format,
	})

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

	apkKeyName, err := t.Apply(overridden.APK.Signature.KeyName)
	if err != nil {
		return err
	}

	libdirs := config.Libdirs{}

	libdirs.Header, err = t.Apply(fpm.Libdirs.Header)
	if err != nil {
		return err
	}
	libdirs.CShared, err = t.Apply(fpm.Libdirs.CShared)
	if err != nil {
		return err
	}
	libdirs.CArchive, err = t.Apply(fpm.Libdirs.CArchive)
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

	if len(fpm.Deb.Lintian) > 0 && (format == "deb" || format == "termux.deb") {
		lintian, err := setupLintian(ctx, fpm, packageName, format, arch)
		if err != nil {
			return err
		}
		contents = append(contents, lintian)
	}

	log := log.WithField("package", packageName).WithField("format", format).WithField("arch", arch)

	// FPM meta package should not contain binaries at all
	if !fpm.Meta {
		for _, art := range artifacts {
			src := art.Path
			dst := filepath.Join(artifactPackageDir(fpm.Bindir, libdirs, art), art.Name)
			log.WithField("src", src).
				WithField("dst", dst).
				WithField("type", art.Type.String()).
				Debug("adding artifact to package")
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
		Arch:            infoArch,
		Platform:        infoPlatform,
		Name:            packageName,
		Version:         ctx.Version,
		Section:         fpm.Section,
		Priority:        fpm.Priority,
		Epoch:           fpm.Epoch,
		Release:         fpm.Release,
		Prerelease:      fpm.Prerelease,
		VersionMetadata: fpm.VersionMetadata,
		Maintainer:      fpm.Maintainer,
		Description:     fpm.Description,
		Vendor:          fpm.Vendor,
		Homepage:        fpm.Homepage,
		License:         fpm.License,
		Changelog:       fpm.Changelog,
		Overridables: nfpm.Overridables{
			Umask:      overridden.Umask,
			Conflicts:  overridden.Conflicts,
			Depends:    overridden.Dependencies,
			Recommends: overridden.Recommends,
			Provides:   overridden.Provides,
			Suggests:   overridden.Suggests,
			Replaces:   overridden.Replaces,
			Contents:   contents,
			Scripts: nfpm.Scripts{
				PreInstall:  overridden.Scripts.PreInstall,
				PostInstall: overridden.Scripts.PostInstall,
				PreRemove:   overridden.Scripts.PreRemove,
				PostRemove:  overridden.Scripts.PostRemove,
			},
			Deb: nfpm.Deb{
				Compression: overridden.Deb.Compression,
				Fields:      overridden.Deb.Fields,
				Predepends:  overridden.Deb.Predepends,
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
						// TODO: Method, Type, KeyID
					},
					Type: overridden.Deb.Signature.Type,
				},
			},
			RPM: nfpm.RPM{
				Summary:     overridden.RPM.Summary,
				Group:       overridden.RPM.Group,
				Compression: overridden.RPM.Compression,
				Prefixes:    overridden.RPM.Prefixes,
				Packager:    overridden.RPM.Packager,
				Signature: nfpm.RPMSignature{
					PackageSignature: nfpm.PackageSignature{
						KeyFile:       rpmKeyFile,
						KeyPassphrase: getPassphraseFromEnv(ctx, "RPM", fpm.ID),
						// TODO: KeyID
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
					KeyName: apkKeyName,
				},
				Scripts: nfpm.APKScripts{
					PreUpgrade:  overridden.APK.Scripts.PreUpgrade,
					PostUpgrade: overridden.APK.Scripts.PostUpgrade,
				},
			},
			ArchLinux: nfpm.ArchLinux{
				Pkgbase:  overridden.ArchLinux.Pkgbase,
				Packager: overridden.ArchLinux.Packager,
				Scripts: nfpm.ArchLinuxScripts{
					PreUpgrade:  overridden.ArchLinux.Scripts.PreUpgrade,
					PostUpgrade: overridden.ArchLinux.Scripts.PostUpgrade,
				},
			},
		},
	}

	if skips.Any(ctx, skips.Sign) {
		info.APK.Signature = nfpm.APKSignature{}
		info.RPM.Signature = nfpm.RPMSignature{}
		info.Deb.Signature = nfpm.DebSignature{}
	}

	packager, err := nfpm.Get(strings.Replace(format, "termux.", "", 1))
	if err != nil {
		return err
	}

	ext := "." + format
	if packager, ok := packager.(nfpm.PackagerWithExtension); ok {
		if format != "termux.deb" {
			ext = packager.ConventionalExtension()
		}
	}

	info = nfpm.WithDefaults(info)
	packageFilename, err := t.WithExtraFields(tmpl.Fields{
		"ConventionalFileName":  packager.ConventionalFileName(info),
		"ConventionalExtension": ext,
	}).Apply(overridden.FileNameTemplate)
	if err != nil {
		return err
	}

	if !strings.HasSuffix(packageFilename, ext) {
		packageFilename += ext
	}

	path := filepath.Join(ctx.Config.Dist, packageFilename)
	log.WithField("file", path).Info("creating")
	w, err := os.Create(path)
	if err != nil {
		return err
	}
	defer w.Close()

	if err := packager.Package(info, w); err != nil {
		return fmt.Errorf("nfpm failed for %s: %w", packageFilename, err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("could not close package file: %w", err)
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Type:    artifact.LinuxPackage,
		Name:    packageFilename,
		Path:    path,
		Goos:    artifacts[0].Goos,
		Goarch:  artifacts[0].Goarch,
		Goarm:   artifacts[0].Goarm,
		Gomips:  artifacts[0].Gomips,
		Goamd64: artifacts[0].Goamd64,
		Extra: map[string]interface{}{
			artifact.ExtraID:     fpm.ID,
			artifact.ExtraFormat: format,
			artifact.ExtraExt:    format,
			extraFiles:           contents,
		},
	})
	return nil
}

func setupLintian(ctx *context.Context, fpm config.NFPM, packageName, format, arch string) (*files.Content, error) {
	lines := make([]string, 0, len(fpm.Deb.Lintian))
	for _, ov := range fpm.Deb.Lintian {
		lines = append(lines, fmt.Sprintf("%s: %s", packageName, ov))
	}
	lintianPath := filepath.Join(ctx.Config.Dist, format, packageName+"_"+arch, "lintian")
	if err := os.MkdirAll(filepath.Dir(lintianPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to write lintian file: %w", err)
	}
	if err := os.WriteFile(lintianPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write lintian file: %w", err)
	}

	log.Debugf("creating %q", lintianPath)
	return &files.Content{
		Source:      lintianPath,
		Destination: "./usr/share/lintian/overrides/" + packageName,
		Packager:    "deb",
		FileInfo: &files.ContentFileInfo{
			Mode: 0o644,
		},
	}, nil
}

func destinations(contents files.Contents) []string {
	result := make([]string, 0, len(contents))
	for _, f := range contents {
		result = append(result, f.Destination)
	}
	return result
}

func getPassphraseFromEnv(ctx *context.Context, packager string, nfpmID string) string {
	nfpmID = strings.ToUpper(nfpmID)
	for _, k := range []string{
		fmt.Sprintf("NFPM_%s_%s_PASSPHRASE", nfpmID, packager),
		fmt.Sprintf("NFPM_%s_PASSPHRASE", nfpmID),
		"NFPM_PASSPHRASE",
	} {
		if v, ok := ctx.Env[k]; ok {
			return v
		}
	}

	return ""
}

func termuxPrefixedDir(dir string) string {
	if dir == "" {
		return ""
	}
	return filepath.Join("/data/data/com.termux/files", dir)
}

func artifactPackageDir(bindir string, libdirs config.Libdirs, art *artifact.Artifact) string {
	switch art.Type {
	case artifact.Binary:
		return bindir
	case artifact.Header:
		return libdirs.Header
	case artifact.CShared:
		return libdirs.CShared
	case artifact.CArchive:
		return libdirs.CArchive
	default:
		// should never happen
		return ""
	}
}
