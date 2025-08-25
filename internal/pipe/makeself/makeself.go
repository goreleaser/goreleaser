// Package makeself implements the Pipe interface providing makeself self-extracting archive support.
package makeself

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/archivefiles"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gio"
	"github.com/goreleaser/goreleaser/v2/internal/ids"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const (
	defaultNameTemplate = `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ with .Arm }}v{{ . }}{{ end }}{{ with .Mips }}_{{ . }}{{ end }}{{ if not (eq .Amd64 "v1") }}{{ .Amd64 }}{{ end }}`
	extraFiles          = "Files"
)

// Pipe for makeself packaging.
type Pipe struct{}

func (Pipe) String() string { return "makeself packages" }
func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Makeself) || len(ctx.Config.Makeselfs) == 0
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("makeselfs")
	for i := range ctx.Config.Makeselfs {
		cfg := &ctx.Config.Makeselfs[i]
		if cfg.ID == "" {
			cfg.ID = "default"
		}
		if cfg.Name == "" {
			cfg.Name = defaultNameTemplate
		}
		if cfg.Extension == "" {
			cfg.Extension = ".run"
		}
		ids.Inc(cfg.ID)
	}
	return ids.Validate()
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for _, cfg := range ctx.Config.Makeselfs {
		return doRun(ctx, cfg)
	}
	return g.Wait()
}

func doRun(ctx *context.Context, cfg config.MakeselfPackage) error {
	filters := []artifact.Filter{
		artifact.Or(
			artifact.ByType(artifact.Binary),
			artifact.ByType(artifact.UniversalBinary),
			artifact.ByType(artifact.Header),
			artifact.ByType(artifact.CArchive),
			artifact.ByType(artifact.CShared),
		),
	}
	if len(cfg.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(cfg.IDs...))
	}
	if len(cfg.Goos) > 0 {
		gf := make([]artifact.Filter, len(cfg.Goos))
		for i, goos := range cfg.Goos {
			gf[i] = artifact.ByGoos(goos)
		}
		filters = append(filters, artifact.Or(gf...))
	}
	if len(cfg.Goarch) > 0 {
		gf := make([]artifact.Filter, len(cfg.Goarch))
		for i, goarch := range cfg.Goarch {
			gf[i] = artifact.ByGoarch(goarch)
		}
		filters = append(filters, artifact.Or(gf...))
	}

	binaries := ctx.Artifacts.
		Filter(artifact.And(filters...)).
		GroupByPlatform()
	if len(binaries) == 0 {
		return fmt.Errorf("no binaries found for builds %v with goos %v goarch %v", cfg.IDs, cfg.Goos, cfg.Goarch)
	}

	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for _, artifacts := range binaries {
		g.Go(func() error {
			return create(ctx, cfg, artifacts)
		})
	}
	return g.Wait()
}

// https://ibiblio.org/pub/linux/LSM-TEMPLATE.html
const lsmTemplate = `Begin4
Title: {{.Title}}
Version: {{.Version}}
Description: {{.Description}}
Keywords: {{.Keywords}}
Author: {{.Maintainer}}
Maintained-by: {{.Maintainer}}
Primary-site: {{.Homepage}}
Platforms: {{.Platform}}
Copying-policy: {{.License}}
End`

func create(ctx *context.Context, cfg config.MakeselfPackage, binaries []*artifact.Artifact) error {
	tpl := tmpl.New(ctx)
	if len(binaries) > 0 {
		tpl = tpl.WithArtifact(binaries[0])
	}

	disable, err := tpl.Bool(cfg.Disable)
	if err != nil {
		return err
	}
	if disable {
		return pipe.Skip("disabled")
	}

	title := cfg.Title
	description := cfg.Description
	packageName := cfg.Name
	maintainer := cfg.Maintainer
	homepage := cfg.Homepage
	license := cfg.License
	label := cfg.Label
	installScript := cfg.InstallScript
	compression := cfg.Compression
	extension := cfg.Extension
	extraArgs := cfg.ExtraArgs
	keywords := cfg.Keywords

	if err := tpl.ApplyAll(
		&title,
		&description,
		&packageName,
		&maintainer,
		&homepage,
		&license,
		&label,
		&installScript,
		&compression,
		&extension,
	); err != nil {
		return err
	}
	if err := tpl.ApplySlice(&extraArgs); err != nil {
		return err
	}
	if err := tpl.ApplySlice(&keywords); err != nil {
		return err
	}

	lsm, err := tpl.WithExtraFields(tmpl.Fields{
		"Title":       title,
		"Description": description,
		"Keywords":    strings.Join(keywords, ", "),
		"Maintainer":  maintainer,
		"Homepage":    homepage,
		"License":     license,
	}).Apply(lsmTemplate)
	if err != nil {
		return err
	}

	// Ensure extension starts with a dot
	if extension != "" && !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}

	packageFilename := packageName + extension
	packagePath := filepath.Join(ctx.Config.Dist, packageFilename)

	log := log.WithField("package", packageName).
		WithField("path", packagePath).
		WithField("extension", extension)
	log.Info("creating makeself package")

	// Create makeself package directly using makeself command
	if err := createMakeselfPackage(
		ctx,
		packagePath,
		binaries,
		cfg,
		tpl,
		label,
		installScript,
		compression,
		lsm,
		extraArgs,
	); err != nil {
		return fmt.Errorf("failed to create makeself package: %w", err)
	}

	bins := []string{}
	for _, binary := range binaries {
		bins = append(bins, binary.Name)
	}

	// Add extra files
	files, err := archivefiles.Eval(tpl, cfg.Files)
	if err != nil {
		return fmt.Errorf("failed to find files to archive: %w", err)
	}

	// Create artifact
	art := &artifact.Artifact{
		Type: artifact.MakeselfPackage,
		Name: packageFilename,
		Path: packagePath,
		Extra: map[string]any{
			artifact.ExtraID:       cfg.ID,
			artifact.ExtraFormat:   "makeself",
			artifact.ExtraExt:      extension,
			artifact.ExtraBinaries: bins,
			extraFiles:             files,
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

func createMakeselfPackage(
	ctx *context.Context,
	packagePath string,
	binaries []*artifact.Artifact,
	cfg config.MakeselfPackage,
	tpl *tmpl.Template,
	label, installScript, compression, lsm string,
	extraArgs []string,
) error {
	// Create the package directory if it doesn't exist
	packageDir := filepath.Dir(packagePath)
	log.Debugf("creating package directory: %s", packageDir)
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", packageDir, err)
	}

	// Verify directory was created
	if _, err := os.Stat(packageDir); os.IsNotExist(err) {
		return fmt.Errorf("package directory %s was not created successfully", packageDir)
	}
	log.Debugf("package directory verified: %s", packageDir)

	// Create temporary directory for files to be archived
	dir, err := os.MkdirTemp("", "makeself-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(dir)

	// Add binaries unless it's a meta package
	for _, binary := range binaries {
		// Preserve binary directory structure by default (matches archive pipeline behavior)
		dst := filepath.Join(dir, filepath.Base(binary.Name))
		// Ensure parent directory exists for nested binary paths
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", binary.Name, err)
		}

		if err := gio.Copy(binary.Path, dst); err != nil {
			return fmt.Errorf("failed to copy binary %s: %w", binary.Name, err)
		}
		// Make binary executable
		if err := os.Chmod(dst, 0o755); err != nil {
			return fmt.Errorf("failed to make binary executable %s: %w", dst, err)
		}
	}

	files, err := archivefiles.Eval(tpl, cfg.Files)
	if err != nil {
		return fmt.Errorf("failed to find files to archive: %w", err)
	}

	for _, f := range files {
		dst := filepath.Join(dir, f.Destination)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", f.Destination, err)
		}
		if err := gio.Copy(f.Source, dst); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", f.Source, err)
		}
	}

	bts, err := os.ReadFile(installScript)
	if err != nil {
		return err
	}
	install, err := tpl.Apply(string(bts))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "install.sh"), []byte(install), 0o755); err != nil {
		return err
	}

	// Create LSM file if LSM template is provided
	lsmFile := filepath.Join(dir, "package.lsm")
	if err := os.WriteFile(lsmFile, []byte(lsm), 0o644); err != nil {
		return fmt.Errorf("failed to write LSM file: %w", err)
	}

	// Build makeself command with configuration
	args := []string{"--quiet"} // Always run quietly

	// Add compression argument
	switch compression {
	case "gzip", "bzip2", "xz", "lzo", "compress":
		args = append(args, "--"+compression)
	case "none":
		args = append(args, "--nocomp")
	default:
		// let makeself choose.
	}

	args = append(args, "--lsm", lsmFile)
	args = append(args, extraArgs...)
	args = append(args, dir, packagePath)
	args = append(args, label)
	args = append(args, "install.sh")

	cmd := exec.CommandContext(ctx, "makeself", args...)

	// TODO: log stdout/err
	// Capture stderr for error reporting
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("makeself failed: %w: %s", err, string(output))
	}
	return nil
}
