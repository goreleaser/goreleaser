// Package config contains the model and loader of the goreleaser configuration
// file.
package config

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v1"
)

// Repo represents any kind of repo (github, gitlab, etc)
type Repo struct {
	Owner string `yaml:",omitempty"`
	Name  string `yaml:",omitempty"`
}

// String of the repo, e.g. owner/name
func (r Repo) String() string {
	return r.Owner + "/" + r.Name
}

// Homebrew contains the brew section
type Homebrew struct {
	GitHub       Repo     `yaml:",omitempty"`
	Folder       string   `yaml:",omitempty"`
	Caveats      string   `yaml:",omitempty"`
	Plist        string   `yaml:",omitempty"`
	Install      string   `yaml:",omitempty"`
	Dependencies []string `yaml:",omitempty"`
	Conflicts    []string `yaml:",omitempty"`
	Description  string   `yaml:",omitempty"`
	Homepage     string   `yaml:",omitempty"`
}

// Hooks define actions to run before and/or after something
type Hooks struct {
	Pre  string `yaml:",omitempty"`
	Post string `yaml:",omitempty"`
}

// IgnoredBuild represents a build ignored by the user
type IgnoredBuild struct {
	Goos, Goarch, Goarm string
}

// Build contains the build configuration section
type Build struct {
	Goos    []string        `yaml:",omitempty"`
	Goarch  []string        `yaml:",omitempty"`
	Goarm   []string        `yaml:",omitempty"`
  Ignore  []IgnoredBuild  `yaml:",omitempty"`
	Main    string          `yaml:",omitempty"`
	Ldflags string          `yaml:",omitempty"`
	Flags   string          `yaml:",omitempty"`
	Binary  string          `yaml:",omitempty"`
	Hooks   Hooks           `yaml:",omitempty"`
}

// FormatOverride is used to specify a custom format for a specific GOOS.
type FormatOverride struct {
	Goos   string `yaml:",omitempty"`
	Format string `yaml:",omitempty"`
}

// Archive config used for the archive
type Archive struct {
	Format          string            `yaml:",omitempty"`
	FormatOverrides []FormatOverride  `yaml:"format_overrides,omitempty"`
	NameTemplate    string            `yaml:"name_template,omitempty"`
	Replacements    map[string]string `yaml:",omitempty"`
	Files           []string          `yaml:",omitempty"`
}

// Release config used for the GitHub release
type Release struct {
	GitHub Repo `yaml:",omitempty"`
	Draft  bool `yaml:",omitempty"`
}

// FPM config
type FPM struct {
	Formats      []string `yaml:",omitempty"`
	Dependencies []string `yaml:",omitempty"`
	Conflicts    []string `yaml:",omitempty"`
	Vendor       string   `yaml:",omitempty"`
	Homepage     string   `yaml:",omitempty"`
	Maintainer   string   `yaml:",omitempty"`
	Description  string   `yaml:",omitempty"`
	License      string   `yaml:",omitempty"`
}

// Snapshot config
type Snapshot struct {
	NameTemplate string `yaml:"name_template,omitempty"`
}

// Project includes all project configuration
type Project struct {
	Release  Release  `yaml:",omitempty"`
	Brew     Homebrew `yaml:",omitempty"`
	Build    Build    `yaml:",omitempty"`
	Archive  Archive  `yaml:",omitempty"`
	FPM      FPM      `yaml:",omitempty"`
	Snapshot Snapshot `yaml:",omitempty"`

	// test only property indicating the path to the dist folder
	Dist string `yaml:"-"`
}

// Load config file
func Load(file string) (config Project, err error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(data, &config)
	return
}
