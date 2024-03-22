// Package srpm implements the Pipe interface building source RPMs.
package srpm

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/goreleaser/nfpm/v2"
	"github.com/goreleaser/nfpm/v2/files"

	_ "github.com/goreleaser/nfpm/v2/rpm" // blank import to register the format
)

var (
	defaultFileNameTemplate     = "{{ .PackageName }}-{{ .Version }}.src.rpm"
	defaultSpecFileNameTemplate = "{{ .PackageName }}.spec"
)

//go:embed spec.tmpl
var defaultSpecTemplate string

// Pipe for source RPMs.
type Pipe struct{}

func (Pipe) String() string { return "source RPMs" }
func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.SRPM) || !ctx.Config.SRPM.Enabled
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	srpm := &ctx.Config.SRPM
	if srpm.ID == "" {
		srpm.ID = "default"
	}
	if srpm.PackageName == "" {
		srpm.PackageName = ctx.Config.ProjectName
	}
	if srpm.FileNameTemplate == "" {
		srpm.FileNameTemplate = defaultFileNameTemplate
	}
	if srpm.SpecFileNameTemplate == "" {
		srpm.SpecFileNameTemplate = defaultSpecFileNameTemplate
	}
	if srpm.SpecTemplate == "" {
		srpm.SpecTemplate = defaultSpecTemplate
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
		return fmt.Errorf("no source archives found")
	} else if len(sourceArchives) > 1 {
		return fmt.Errorf("multiple source archives found")
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

	// Generate the spec file.
	specFileName, err := t.Apply(srpm.SpecFileNameTemplate)
	if err != nil {
		return err
	}
	specContents, err := t.Apply(srpm.SpecTemplate)
	if err != nil {
		return err
	}
	specPath := filepath.Join(ctx.Config.Dist, specFileName)
	if err := os.WriteFile(specPath, []byte(specContents), 0o666); err != nil {
		return err
	}
	specFileArtifact := &artifact.Artifact{
		Type: artifact.RPMSpec,
		Name: specFileName,
		Path: specPath,
	}

	// Default file info.
	owner := "mockbuild"
	group := "mock"
	mtime := ctx.Git.CommitDate

	contents := files.Contents{}

	// Add the source archive.
	contents = append(contents, &files.Content{
		Source:      sourceArchive.Path,
		Destination: sourceArchive.Name,
		// FIXME Type:
		Packager: srpm.Packager,
		FileInfo: &files.ContentFileInfo{
			Owner: owner,
			Group: group,
			Mode:  0o664, // Source archives are group-writeable by default.
			MTime: mtime,
			// FIXME Size:
		},
	})

	// Add extra contents.
	contents = append(contents, srpm.Contents...)

	// Add the spec file.
	contents = append(contents, &files.Content{
		Source:      specFileArtifact.Path,
		Destination: specFileArtifact.Name,
		// FIXME Type:
		Packager: srpm.Packager,
		FileInfo: &files.ContentFileInfo{
			Owner: owner,
			Group: group,
			Mode:  0o660, // Spec files are private by default.
			MTime: mtime,
			Size:  int64(len(specContents)),
		},
	})

	keyFile, err := t.Apply(srpm.Signature.KeyFile)
	if err != nil {
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
						KeyFile:       keyFile,
						KeyPassphrase: ctx.Env[fmt.Sprintf("SRPM_%s_PASSPHRASE", srpm.ID)],
						// TODO: KeyID
					},
				},
			},
		},
	}

	if skips.Any(ctx, skips.Sign) {
		info.RPM.Signature = nfpm.RPMSignature{}
	}

	packager, err := nfpm.Get("rpm")
	if err != nil {
		return err
	}
	info = nfpm.WithDefaults(info)

	// Write the source RPM.
	srpmFileName, err := t.Apply(srpm.FileNameTemplate)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(srpmFileName, ".src.rpm") {
		srpmFileName += ".src.rpm"
	}
	srpmPath := filepath.Join(ctx.Config.Dist, srpmFileName)
	log.WithField("file", srpmPath).Info("creating")
	srpmFile, err := os.Create(srpmPath)
	if err != nil {
		return err
	}
	defer srpmFile.Close()
	if err := packager.Package(info, srpmFile); err != nil {
		return fmt.Errorf("nfpm failed: %w", err)
	}
	if err := srpmFile.Close(); err != nil {
		return fmt.Errorf("could not close package file: %w", err)
	}
	srpmArtifact := &artifact.Artifact{
		Type: artifact.SourceRPM,
		Name: srpmFileName,
		Path: srpmPath,
	}

	ctx.Artifacts.Add(specFileArtifact)
	ctx.Artifacts.Add(srpmArtifact)
	return nil
}
