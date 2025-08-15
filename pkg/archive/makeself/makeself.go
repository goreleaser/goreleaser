// Package makeself implements the Archive interface providing makeself self-extracting
// archive creation.
package makeself

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
)

// Archive as makeself.
type Archive struct {
	tempDir    string
	files      map[string]bool
	target     io.Writer
	outputPath string // Path where makeself should create the archive directly
	closed     bool
	config     MakeselfConfig // Configuration options for makeself
}

// MakeselfConfig holds configuration options for makeself archives.
type MakeselfConfig struct {
	OutputPath        string   // Optional: override output path
	InstallScript     string   // Optional: custom install script content
	InstallScriptFile string   // Optional: path to custom install script file (relative to archive contents)
	Label             string   // Optional: custom label for the archive
	Compression       string   // Optional: compression format (gzip, bzip2, xz, lzo, compress, none)
	ExtraArgs         []string // Optional: extra command line arguments
	LSMTemplate       string   // Optional: LSM file content template
	LSMFile           string   // Optional: path to external LSM file
}

// New makeself archive.
func New(target io.Writer) *Archive {
	tempDir, err := os.MkdirTemp("", "makeself-*")
	if err != nil {
		panic(fmt.Sprintf("failed to create temp directory: %v", err))
	}

	// Target must be a file for makeself to work
	file, ok := target.(*os.File)
	if !ok {
		panic("makeself archives require an *os.File target")
	}

	return &Archive{
		tempDir:    tempDir,
		files:      map[string]bool{},
		target:     target,
		outputPath: file.Name(),
		closed:     false,
	}
}

// NewWithInstallScript creates a makeself archive with a custom install script.
func NewWithInstallScript(target io.Writer, installScript string) *Archive {
	archive := New(target)

	// Write custom install script
	installPath := filepath.Join(archive.tempDir, "install.sh")
	if err := os.WriteFile(installPath, []byte(installScript), 0o755); err != nil {
		panic(fmt.Sprintf("failed to create install script: %v", err))
	}

	return archive
}

// NewWithConfig creates a makeself archive with full configuration options.
func NewWithConfig(target io.Writer, outputPath string, cfg MakeselfConfig) *Archive {
	archive := New(target)

	// Override output path if provided
	if cfg.OutputPath != "" {
		archive.outputPath = cfg.OutputPath
	} else if outputPath != "" {
		archive.outputPath = outputPath
	}

	// Handle install script - defer validation for file-based scripts
	if cfg.InstallScript != "" {
		// Use provided script content
		installPath := filepath.Join(archive.tempDir, "install.sh")
		if err := os.WriteFile(installPath, []byte(cfg.InstallScript), 0o755); err != nil {
			panic(fmt.Sprintf("failed to create install script: %v", err))
		}
	}

	// Store configuration for use in Close()
	archive.config = cfg

	return archive
}

// Close creates the makeself archive and writes it to the target.
func (a *Archive) Close() error {
	if a.closed {
		return nil // Idempotent close
	}
	a.closed = true

	defer os.RemoveAll(a.tempDir)

	// Check if makeself command is available
	makeselfCmd := findMakeselfCommand()
	if makeselfCmd == "" {
		return fmt.Errorf("makeself command not found in PATH (tried 'makeself' and 'makeself.sh')")
	}

	// Determine the install script to use
	var installScriptArg string
	if a.config.InstallScriptFile != "" {
		// Install script file path is relative to archive contents
		scriptPath := filepath.Join(a.tempDir, a.config.InstallScriptFile)
		if _, err := os.Stat(scriptPath); err != nil {
			return fmt.Errorf("install script file %s not found in archive contents: %w", a.config.InstallScriptFile, err)
		}
		// Use relative path for makeself command, avoid double ./ prefix
		if strings.HasPrefix(a.config.InstallScriptFile, "./") {
			installScriptArg = a.config.InstallScriptFile
		} else {
			installScriptArg = "./" + a.config.InstallScriptFile
		}
	} else {
		// Create a basic install script if none exists
		installScript := filepath.Join(a.tempDir, "install.sh")
		if _, err := os.Stat(installScript); os.IsNotExist(err) {
			installContent := `#!/bin/bash
# Default installation script for makeself archive
# This script is executed after extraction

# Make binaries executable
find . -type f -perm -u+x -exec chmod +x {} \;

echo "Archive extracted successfully to $(pwd)"
echo "Files:"
find . -type f | sort
`
			if err := os.WriteFile(installScript, []byte(installContent), 0o755); err != nil {
				return fmt.Errorf("failed to create install script: %w", err)
			}
		}
		installScriptArg = "./install.sh"
	}

	// Prepare the output file for makeself
	outputPath := a.outputPath

	// Truncate the target file to prepare for makeself to write to it
	file := a.target.(*os.File)
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate target file: %w", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek to beginning of file: %w", err)
	}

	// Build makeself command with configuration
	args := []string{"--quiet"} // Always run quietly

	// Apply compression setting
	switch strings.ToLower(a.config.Compression) {
	case "none":
		args = append(args, "--nocomp")
	case "gzip", "gz":
		args = append(args, "--gzip")
	case "bzip2", "bz2":
		args = append(args, "--bzip2")
	case "xz":
		args = append(args, "--xz")
	case "lzo":
		args = append(args, "--lzo")
	case "compress":
		args = append(args, "--compress")
	case "":
		// Default: let makeself choose its default (usually gzip)
	default:
		// For unknown compression types, log a warning but continue
		fmt.Fprintf(os.Stderr, "Warning: unknown compression format '%s', using makeself default\n", a.config.Compression)
	}

	// Handle LSM configuration
	var lsmFile string
	var cleanupLSM bool
	if a.config.LSMTemplate != "" {
		// Create temporary LSM file from template
		tmpFile, err := os.CreateTemp("", "lsm-*.txt")
		if err != nil {
			return fmt.Errorf("failed to create temporary LSM file: %w", err)
		}
		lsmFile = tmpFile.Name()
		cleanupLSM = true

		if _, err := tmpFile.WriteString(a.config.LSMTemplate); err != nil {
			tmpFile.Close()
			os.Remove(lsmFile)
			return fmt.Errorf("failed to write LSM template to file: %w", err)
		}
		tmpFile.Close()
	} else if a.config.LSMFile != "" {
		// Use external LSM file
		lsmFile = a.config.LSMFile
		// Verify the file exists
		if _, err := os.Stat(lsmFile); err != nil {
			return fmt.Errorf("LSM file %s not found: %w", lsmFile, err)
		}
	}

	// Clean up temporary LSM file when done
	if cleanupLSM {
		defer os.Remove(lsmFile)
	}

	// Add LSM file argument if specified
	if lsmFile != "" {
		args = append(args, "--lsm", lsmFile)
	}

	// Add any extra arguments from configuration
	args = append(args, a.config.ExtraArgs...)

	// Add required positional arguments
	args = append(args, a.tempDir)  // Source directory
	args = append(args, outputPath) // Output file

	// Use custom label or default
	label := "Self-extracting archive"
	if a.config.Label != "" {
		label = a.config.Label
	}
	args = append(args, label)

	// Use the determined install script argument
	args = append(args, installScriptArg)

	// Create the makeself archive command
	cmd := exec.Command(makeselfCmd, args...)

	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("makeself failed: %w: %s", err, stderr.String())
	}

	// Make the archive executable like makeself normally does
	if info, err := file.Stat(); err == nil {
		// Add executable permission for user, group, and other
		newMode := info.Mode() | 0o111
		if err := file.Chmod(newMode); err != nil {
			// Don't fail if we can't set permissions - just log it
			fmt.Fprintf(os.Stderr, "Warning: failed to make makeself archive executable: %v\n", err)
		}
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
	// Not found
	return ""
}

// Add file to the archive.
func (a *Archive) Add(f config.File) error {
	if a.closed {
		return fmt.Errorf("cannot add files to closed archive")
	}
	if _, ok := a.files[f.Destination]; ok {
		return fmt.Errorf("file %s already exists in archive", f.Destination)
	}

	destPath := filepath.Join(a.tempDir, f.Destination)

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(destPath), err)
	}

	// Copy file to temp directory
	src, err := os.Open(f.Source)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", f.Source, err)
	}
	defer src.Close()

	srcInfo, err := src.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file %s: %w", f.Source, err)
	}

	if srcInfo.IsDir() {
		return fmt.Errorf("directories are not supported in makeself archives: %s", f.Source)
	}

	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", destPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy file %s to %s: %w", f.Source, destPath, err)
	}

	// Set file permissions
	mode := srcInfo.Mode()
	if f.Info.Mode != 0 {
		mode = f.Info.Mode
	}
	if err := os.Chmod(destPath, mode); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", destPath, err)
	}

	// Set modification time
	if !f.Info.ParsedMTime.IsZero() {
		if err := os.Chtimes(destPath, f.Info.ParsedMTime, f.Info.ParsedMTime); err != nil {
			return fmt.Errorf("failed to set modification time on %s: %w", destPath, err)
		}
	}

	a.files[f.Destination] = true
	return nil
}
