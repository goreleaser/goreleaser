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
		makeself := &ctx.Config.Makeselfs[i]
		if makeself.ID == "" {
			makeself.ID = "default"
		}
		if makeself.NameTemplate == "" {
			makeself.NameTemplate = defaultNameTemplate
		}
		if makeself.Extension == "" {
			makeself.Extension = ".run"
		}
		ids.Inc(makeself.ID)
	}
	return ids.Validate()
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	skips := pipe.SkipMemento{}
	for _, makeself := range ctx.Config.Makeselfs {
		err := doRun(ctx, makeself)
		if pipe.IsSkip(err) {
			skips.Remember(err)
			continue
		}
		if err != nil {
			return err
		}
	}
	return skips.Evaluate()
}

func doRun(ctx *context.Context, makeselfCfg config.MakeselfPackage) error {
	// Handle meta packages - they don't need binaries
	if makeselfCfg.Meta {
		return create(ctx, makeselfCfg, nil)
	}

	filters := []artifact.Filter{
		artifact.Or(
			artifact.ByType(artifact.Binary),
			artifact.ByType(artifact.UniversalBinary),
			artifact.ByType(artifact.Header),
			artifact.ByType(artifact.CArchive),
			artifact.ByType(artifact.CShared),
		),
	}
	if len(makeselfCfg.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(makeselfCfg.IDs...))
	}

	// Add platform filtering
	log.Debugf("makeself config %s: goos=%v, goarch=%v", makeselfCfg.ID, makeselfCfg.Goos, makeselfCfg.Goarch)
	if len(makeselfCfg.Goos) > 0 {
		goosFilters := make([]artifact.Filter, len(makeselfCfg.Goos))
		for i, goos := range makeselfCfg.Goos {
			goosFilters[i] = artifact.ByGoos(goos)
		}
		filters = append(filters, artifact.Or(goosFilters...))
		log.Debugf("makeself config %s: added goos filtering for %v", makeselfCfg.ID, makeselfCfg.Goos)
	} else {
		log.Warnf("makeself config %s: NO goos filtering - will process ALL platforms!", makeselfCfg.ID)
	}
	if len(makeselfCfg.Goarch) > 0 {
		goarchFilters := make([]artifact.Filter, len(makeselfCfg.Goarch))
		for i, goarch := range makeselfCfg.Goarch {
			goarchFilters[i] = artifact.ByGoarch(goarch)
		}
		filters = append(filters, artifact.Or(goarchFilters...))
		log.Debugf("makeself config %s: added goarch filtering for %v", makeselfCfg.ID, makeselfCfg.Goarch)
	}

	binaries := ctx.Artifacts.
		Filter(artifact.And(filters...)).
		GroupByPlatform()
	if len(binaries) == 0 {
		return fmt.Errorf("no binaries found for builds %v with goos %v goarch %v", makeselfCfg.IDs, makeselfCfg.Goos, makeselfCfg.Goarch)
	}

	g := semerrgroup.New(ctx.Parallelism)
	for _, artifacts := range binaries {
		g.Go(func() error {
			return create(ctx, makeselfCfg, artifacts)
		})
	}
	return g.Wait()
}

func create(ctx *context.Context, makeselfCfg config.MakeselfPackage, binaries []*artifact.Artifact) error {
	template := tmpl.New(ctx)
	if len(binaries) > 0 {
		template = template.WithArtifact(binaries[0])
	}

	// Check if disabled
	if makeselfCfg.Disable != "" {
		disable, err := template.Apply(makeselfCfg.Disable)
		if err != nil {
			return err
		}
		if disable == "true" {
			return nil // Return nil instead of skip error for disabled packages
		}
	}

	var packageName string
	var err error

	// For meta packages without binaries, use a simpler name template if the default is being used
	if makeselfCfg.Meta && len(binaries) == 0 && makeselfCfg.NameTemplate == defaultNameTemplate {
		// Use a meta-package friendly name template
		metaTemplate := `{{ .ProjectName }}_{{ .Version }}_meta`
		packageName, err = template.Apply(metaTemplate)
	} else {
		packageName, err = template.Apply(makeselfCfg.NameTemplate)
	}
	if err != nil {
		return err
	}

	// Apply templates to all configuration fields
	label := makeselfCfg.Label
	if label != "" {
		label, err = template.Apply(label)
		if err != nil {
			return fmt.Errorf("failed to apply template to label: %w", err)
		}
	}

	installScript := makeselfCfg.InstallScript
	if installScript != "" {
		installScript, err = template.Apply(installScript)
		if err != nil {
			return fmt.Errorf("failed to apply template to install_script: %w", err)
		}
	}

	installScriptFile := makeselfCfg.InstallScriptFile
	if installScriptFile != "" {
		installScriptFile, err = template.Apply(installScriptFile)
		if err != nil {
			return fmt.Errorf("failed to apply template to install_script_file: %w", err)
		}
	}

	compression := makeselfCfg.Compression
	if compression != "" {
		compression, err = template.Apply(compression)
		if err != nil {
			return fmt.Errorf("failed to apply template to compression: %w", err)
		}
	}

	lsmTemplate := makeselfCfg.LSMTemplate
	if lsmTemplate != "" {
		lsmTemplate, err = template.Apply(lsmTemplate)
		if err != nil {
			return fmt.Errorf("failed to apply template to lsm_template: %w", err)
		}
	}

	lsmFile := makeselfCfg.LSMFile
	if lsmFile != "" {
		lsmFile, err = template.Apply(lsmFile)
		if err != nil {
			return fmt.Errorf("failed to apply template to lsm_file: %w", err)
		}
	}

	extension := makeselfCfg.Extension
	if extension != "" {
		extension, err = template.Apply(extension)
		if err != nil {
			return fmt.Errorf("failed to apply template to extension: %w", err)
		}
	}

	// Ensure we always have an extension - fallback to .run if empty
	if extension == "" {
		extension = ".run"
	}

	// Apply templates to extra args
	extraArgs := make([]string, len(makeselfCfg.ExtraArgs))
	for i, arg := range makeselfCfg.ExtraArgs {
		extraArgs[i], err = template.Apply(arg)
		if err != nil {
			return fmt.Errorf("failed to apply template to extra_args[%d]: %w", i, err)
		}
	}

	// Ensure extension starts with a dot
	if extension != "" && !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}

	packageFilename := packageName + extension
	packagePath := filepath.Join(ctx.Config.Dist, packageFilename)

	log := log.WithField("package", packageName).WithField("path", packagePath).WithField("extension", extension)
	log.Info("creating makeself package")

	// Ensure the filename has the correct extension for debugging
	log.Debugf("Final package filename: %s (name=%s, ext=%s)", packageFilename, packageName, extension)

	// Create makeself package directly using makeself command
	err = createMakeselfPackage(ctx, packagePath, binaries, makeselfCfg, template, label, installScript, installScriptFile, compression, lsmTemplate, lsmFile, extraArgs)
	if err != nil {
		return fmt.Errorf("failed to create makeself package: %w", err)
	}

	bins := []string{}
	if !makeselfCfg.Meta {
		for _, binary := range binaries {
			bins = append(bins, binary.Name)
			log.WithField("binary", binary.Name).Debug("added binary to makeself package")
		}
	}

	// Add extra files
	files, err := archivefiles.Eval(template, makeselfCfg.Files)
	if err != nil {
		return fmt.Errorf("failed to find files to archive: %w", err)
	}

	// Create artifact
	art := &artifact.Artifact{
		Type: artifact.MakeselfPackage,
		Name: packageFilename,
		Path: packagePath,
		Extra: map[string]any{
			artifact.ExtraID:       makeselfCfg.ID,
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

func createMakeselfPackage(ctx *context.Context, packagePath string, binaries []*artifact.Artifact, makeselfCfg config.MakeselfPackage, template *tmpl.Template, label, installScript, installScriptFile, compression, lsmTemplate, lsmFile string, extraArgs []string) error {
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

	// Check if makeself command is available
	makeselfCmd := findMakeselfCommand()
	if makeselfCmd == "" {
		return fmt.Errorf("makeself command not found in PATH (tried 'makeself' and 'makeself.sh')")
	}

	// Create temporary directory for files to be archived
	tempDir, err := os.MkdirTemp("", "makeself-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Add binaries unless it's a meta package
	if !makeselfCfg.Meta {
		for _, binary := range binaries {
			// Preserve binary directory structure by default (matches archive pipeline behavior)
			var dst string
			if makeselfCfg.StripBinaryDirectory {
				dst = filepath.Join(tempDir, filepath.Base(binary.Name))
			} else {
				dst = filepath.Join(tempDir, binary.Name)
				// Ensure parent directory exists for nested binary paths
				if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
					return fmt.Errorf("failed to create directory for %s: %w", binary.Name, err)
				}
			}

			if err := copyFile(binary.Path, dst); err != nil {
				return fmt.Errorf("failed to copy binary %s: %w", binary.Name, err)
			}
			// Make binary executable
			if err := os.Chmod(dst, 0o755); err != nil {
				return fmt.Errorf("failed to make binary executable %s: %w", dst, err)
			}
		}
	}

	// Add extra files
	files, err := archivefiles.Eval(template, makeselfCfg.Files)
	if err != nil {
		return fmt.Errorf("failed to find files to archive: %w", err)
	}
	if makeselfCfg.Meta && len(files) == 0 {
		return fmt.Errorf("no files found for meta package")
	}

	for _, f := range files {
		dst := filepath.Join(tempDir, f.Destination)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", f.Destination, err)
		}
		if err := copyFile(f.Source, dst); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", f.Source, err)
		}
	}

	// Create install script if provided
	var installScriptPath string
	if installScript != "" {
		installScriptPath = filepath.Join(tempDir, "install.sh")
		if err := os.WriteFile(installScriptPath, []byte(installScript), 0o755); err != nil {
			return fmt.Errorf("failed to write install script: %w", err)
		}
		installScriptFile = "install.sh"
	} else if installScriptFile != "" {
		// Use relative path for makeself command, avoid double ./ prefix
		if strings.HasPrefix(installScriptFile, "./") {
			installScriptFile = installScriptFile[2:]
		}
	} else {
		// Create default install script
		installScriptPath = filepath.Join(tempDir, "install.sh")
		installContent := `#!/bin/bash
# Default installation script for makeself archive
# This script is executed after extraction
echo "Files extracted to: $PWD"
echo "Installation complete."
`
		if err := os.WriteFile(installScriptPath, []byte(installContent), 0o755); err != nil {
			return fmt.Errorf("failed to write default install script: %w", err)
		}
		installScriptFile = "install.sh"
	}

	// Create LSM file if LSM template is provided
	var lsmFilePath string
	if lsmTemplate != "" {
		lsmFilePath = filepath.Join(tempDir, "package.lsm")
		if err := os.WriteFile(lsmFilePath, []byte(lsmTemplate), 0o644); err != nil {
			return fmt.Errorf("failed to write LSM file: %w", err)
		}
		lsmFile = lsmFilePath
	}

	// Build makeself command with configuration
	args := []string{"--quiet"} // Always run quietly

	// Add compression argument
	switch compression {
	case "gzip":
		args = append(args, "--gzip")
	case "bzip2":
		args = append(args, "--bzip2")
	case "xz":
		args = append(args, "--xz")
	case "lzo":
		args = append(args, "--lzo")
	case "compress":
		args = append(args, "--compress")
	case "none":
		args = append(args, "--nocomp")
	case "":
		// Default: let makeself choose its default (usually gzip)
	default:
		// For unknown compression types, log a warning but continue
		log.Warnf("unknown compression format '%s', using makeself default", compression)
	}

	// Add LSM file if specified
	if lsmFile != "" {
		args = append(args, "--lsm", lsmFile)
	}

	// Add extra arguments
	for _, arg := range extraArgs {
		if arg != "" {
			args = append(args, arg)
		}
	}

	// Add required arguments: directory, package, label, startup_script
	args = append(args, tempDir, packagePath)

	if label == "" {
		label = "Self-extracting archive"
	}
	args = append(args, label)

	if installScriptFile != "" {
		args = append(args, installScriptFile)
	}

	// Create the makeself archive command - execute from original working directory
	cmd := exec.Command(makeselfCmd, args...)

	// Debug: Log the full command being executed
	log.Debugf("executing makeself command: %s %v", makeselfCmd, args)

	// Capture stderr for error reporting
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("makeself failed: %w: %s", err, string(output))
	}

	// Debug: Log makeself output and check if file was created
	log.Debugf("makeself output: %s", string(output))
	_, err = os.Stat(packagePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("makeself command completed but output file was not created at %s", packagePath)
	}

	// Stat failed for some other reason
	if err != nil {
		return fmt.Errorf("makeself command completed but stat on file %s failed with: %w", packagePath, err)
	}

	// Make the archive executable
	if err := os.Chmod(packagePath, 0o755); err != nil {
		// Don't fail if we can't set permissions - just log it
		log.Warnf("failed to make makeself archive executable: %v", err)
	}

	return nil
}

// findMakeselfCommand finds the makeself command in PATH, trying both 'makeself' and 'makeself.sh'
func findMakeselfCommand() string {
	// Try 'makeself' first (common on some distributions)
	if _, err := exec.LookPath("makeself"); err == nil {
		return "makeself"
	}
	// Try 'makeself.sh' (traditional name)
	if _, err := exec.LookPath("makeself.sh"); err == nil {
		return "makeself.sh"
	}
	return ""
}

// copyFile copies a file from src to dst, preserving permissions
func copyFile(src, dst string) error {
	// Get source file info to preserve permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer output.Close()

	_, err = output.ReadFrom(input)
	if err != nil {
		return err
	}

	// Preserve source file permissions
	return os.Chmod(dst, srcInfo.Mode())
}
