// Package srpm implements the Pipe interface building source RPMs.
package srpm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/goreleaser/nfpm/v2"
	"github.com/goreleaser/nfpm/v2/files"

	_ "github.com/goreleaser/nfpm/v2/rpm" // blank import to register the srpm packager
)

const (
	extension               = ".src.rpm"
	defaultFileNameTemplate = "{{ .PackageName }}-{{ .Version }}" + extension
)

// Pipe for source RPMs.
type Pipe struct{}

func (Pipe) String() string { return "source RPM" }
func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.SRPM) || !ctx.Config.SRPM.Enabled
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	srpm := &ctx.Config.SRPM
	if srpm.PackageName == "" {
		srpm.PackageName = ctx.Config.ProjectName
	}
	if srpm.FileNameTemplate == "" {
		srpm.FileNameTemplate = defaultFileNameTemplate
	}
	if srpm.Bins == nil {
		srpm.Bins = map[string]string{
			ctx.Config.ProjectName: "%{goipath}",
		}
	}
	return nil
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	sourceArchives := ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableSourceArchive)).List()
	if len(sourceArchives) == 0 {
		return errors.New("no source archives found")
	} else if len(sourceArchives) > 1 {
		return errors.New("multiple source archives found")
	}

	srpm := ctx.Config.SRPM
	sourceArchive := sourceArchives[0]

	t := tmpl.New(ctx).
		WithExtraFields(tmpl.Fields{
			"Summary":         srpm.Summary,
			"Group":           srpm.Group,
			"PackageName":     srpm.PackageName,
			"Epoch":           srpm.Epoch,
			"Section":         srpm.Section,
			"Maintainer":      srpm.Maintainer,
			"Vendor":          srpm.Vendor,
			"Packager":        srpm.Packager,
			"ImportPath":      srpm.ImportPath,
			"License":         srpm.License,
			"LicenseFileName": srpm.LicenseFileName,
			"URL":             srpm.URL,
			"Description":     srpm.Description,
			"Source":          sourceArchive.Name,
			"Bins":            srpm.Bins,
			"Docs":            srpm.Docs,
		})

	// Get the spec template.
	specTemplate, err := os.ReadFile(srpm.SpecFile)
	if err != nil {
		return err
	}
	specContents, err := t.Apply(string(specTemplate))
	if err != nil {
		return err
	}
	specPath := filepath.Join(ctx.Config.Dist, srpm.PackageName+".srpm.spec")
	if err := os.MkdirAll(filepath.Dir(specPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(specPath, []byte(specContents), 0o666); err != nil {
		return err
	}

	// Default file info.
	owner := "mockbuild"
	group := "mock"
	mtime := ctx.Git.CommitDate

	contents := files.Contents{}

	// Add the source archive.
	sourceArchiveFileInfo, err := os.Stat(sourceArchive.Path)
	if err != nil {
		return err
	}
	contents = append(contents, &files.Content{
		Source:      sourceArchive.Path,
		Destination: sourceArchive.Name,
		FileInfo: &files.ContentFileInfo{
			Owner: owner,
			Group: group,
			Mode:  0o664, // Source archives are group-writeable by default.
			MTime: mtime,
			Size:  sourceArchiveFileInfo.Size(),
		},
	})

	// Add extra contents.
	for _, content := range srpm.Contents {
		if err := t.ApplyAll(
			&content.Source,
			&content.Destination,
			&content.FileInfo.Owner,
			&content.FileInfo.Group,
			&content.FileInfo.MTime,
		); err != nil {
			return err
		}
		if content.FileInfo.MTime != "" {
			var err error
			content.FileInfo.ParsedMTime, err = time.Parse(time.RFC3339Nano, content.FileInfo.MTime)
			if err != nil {
				return fmt.Errorf("failed to parse %s: %w", content.FileInfo.MTime, err)
			}
		}
		contents = append(contents, &files.Content{
			Source:      filepath.ToSlash(content.Source),
			Destination: filepath.ToSlash(content.Destination),
			Type:        content.Type,
			Packager:    content.Packager,
			FileInfo: &files.ContentFileInfo{
				Owner: content.FileInfo.Owner,
				Group: content.FileInfo.Group,
				Mode:  content.FileInfo.Mode,
				MTime: content.FileInfo.ParsedMTime,
			},
		})
	}

	// Add the spec file.
	contents = append(contents, &files.Content{
		Source:      specPath,
		Destination: filepath.Base(specPath),
		FileInfo: &files.ContentFileInfo{
			Owner: owner,
			Group: group,
			Mode:  0o660, // Spec files are private by default.
			MTime: mtime,
			Size:  int64(len(specContents)),
		},
	})

	if err := t.ApplyAll(
		&srpm.Signature.KeyFile,
	); err != nil {
		return err
	}

	// Create the source RPM package.
	info := &nfpm.Info{
		Name:        srpm.PackageName,
		Epoch:       srpm.Epoch,
		Version:     ctx.Version,
		Section:     srpm.Section,
		Maintainer:  srpm.Maintainer,
		Description: srpm.Description,
		Vendor:      srpm.Vendor,
		Homepage:    srpm.URL,
		License:     srpm.License,
		Overridables: nfpm.Overridables{
			Contents: contents,
			RPM: nfpm.RPM{
				Group:       srpm.Group,
				Summary:     srpm.Summary,
				Compression: srpm.Compression,
				Packager:    srpm.Packager,
				Signature: nfpm.RPMSignature{
					PackageSignature: nfpm.PackageSignature{
						KeyFile:       srpm.Signature.KeyFile,
						KeyPassphrase: ctx.Env["SRPM_PASSPHRASE"],
					},
				},
			},
		},
	}

	if skips.Any(ctx, skips.Sign) {
		info.RPM.Signature = nfpm.RPMSignature{}
	}

	packager, err := nfpm.Get("srpm")
	if err != nil {
		return err
	}
	info = nfpm.WithDefaults(info)

	packageFilename, err := t.WithExtraFields(tmpl.Fields{
		"ConventionalFileName":  packager.ConventionalFileName(info),
		"ConventionalExtension": extension,
	}).Apply(srpm.FileNameTemplate)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(packageFilename, extension) {
		packageFilename += extension
	}

	// Write the source RPM.
	srpmPath := filepath.Join(ctx.Config.Dist, packageFilename)
	log.WithField("file", srpmPath).Info("creating")
	srpmFile, err := os.Create(srpmPath)
	if err != nil {
		return err
	}
	if err := packager.Package(info, srpmFile); err != nil {
		srpmFile.Close()
		return fmt.Errorf("nfpm failed: %w", err)
	}
	if err := srpmFile.Close(); err != nil {
		return fmt.Errorf("could not close package file: %w", err)
	}

	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.SourceRPM,
		Name: packageFilename,
		Path: srpmPath,
		Extra: map[string]any{
			artifact.ExtraFormat: strings.TrimPrefix(extension, "."),
			artifact.ExtraExt:    extension,
		},
	})
	return nil
}
