// Package config contains makeself-specific configuration structures.
package config

// MakeselfPackage represents a makeself self-extracting archive configuration.
type MakeselfPackage struct {
	ID           string   `yaml:"id,omitempty" json:"id,omitempty"`
	IDs          []string `yaml:"ids,omitempty" json:"ids,omitempty"`
	NameTemplate string   `yaml:"name_template,omitempty" json:"name_template,omitempty"`

	// Makeself-specific configuration
	Label             string   `yaml:"label,omitempty" json:"label,omitempty"`
	InstallScript     string   `yaml:"install_script,omitempty" json:"install_script,omitempty"`
	InstallScriptFile string   `yaml:"install_script_file,omitempty" json:"install_script_file,omitempty"`
	Compression       string   `yaml:"compression,omitempty" json:"compression,omitempty"`
	ExtraArgs         []string `yaml:"extra_args,omitempty" json:"extra_args,omitempty"`
	LSMTemplate       string   `yaml:"lsm_template,omitempty" json:"lsm_template,omitempty"`
	LSMFile           string   `yaml:"lsm_file,omitempty" json:"lsm_file,omitempty"`
	Extension         string   `yaml:"extension,omitempty" json:"extension,omitempty"`

	// Binary path handling - matches archive pipeline behavior
	StripBinaryDirectory bool `yaml:"strip_binary_directory,omitempty" json:"strip_binary_directory,omitempty"`

	// Platform filtering
	Goos   []string `yaml:"goos,omitempty" json:"goos,omitempty"`
	Goarch []string `yaml:"goarch,omitempty" json:"goarch,omitempty"`

	// Files to include in the package (in addition to binaries)
	Files []File `yaml:"files,omitempty" json:"files,omitempty"`

	// Meta package - no binaries, only files
	Meta bool `yaml:"meta,omitempty" json:"meta,omitempty"`

	// Control options
	Disable string `yaml:"disable,omitempty" json:"disable,omitempty" jsonschema:"oneof_type=string;boolean"`
}
