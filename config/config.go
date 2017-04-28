// Package config contains the model and loader of the goreleaser configuration
// file.
package config

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v1"
)

// Repo represents any kind of repo (github, gitlab, etc)
type Repo struct {
	Owner string `yaml:"owner,omitempty"`
	Name  string `yaml:"name,omitempty"`
}

// String of the repo, e.g. owner/name
func (r Repo) String() string {
	return r.Owner + "/" + r.Name
}

// Homebrew contains the brew section
type Homebrew struct {
	GitHub       Repo     `yaml:"github,omitempty"`
	Folder       string   `yaml:"folder,omitempty"`
	Caveats      string   `yaml:"caveats,omitempty"`
	Plist        string   `yaml:"plist,omitempty"`
	Install      string   `yaml:"install,omitempty"`
	Dependencies []string `yaml:"dependencies,omitempty"`
	Conflicts    []string `yaml:"conflicts,omitempty"`
	Description  string   `yaml:"description,omitempty"`
	Homepage     string   `yaml:"homepage,omitempty"`
}

// Hooks define actions to run before and/or after something
type Hooks struct {
	Pre  string `yaml:"pre,omitempty"`
	Post string `yaml:"post,omitempty"`
}

// Build contains the build configuration section
type Build struct {
	Goos    []string `yaml:"goos,omitempty"`
	Goarch  []string `yaml:"goarch,omitempty"`
	Goarm   []string `yaml:"goarm,omitempty"`
	Main    string   `yaml:"main,omitempty"`
	Ldflags string   `yaml:"ldflags,omitempty"`
	Flags   string   `yaml:"flags,omitempty"`
	Binary  string   `yaml:"binary,omitempty"`
	Hooks   Hooks    `yaml:"hooks,omitempty"`
}

// FormatOverride is used to specify a custom format for a specific GOOS.
type FormatOverride struct {
	Goos   string `yaml:"goos,omitempty"`
	Format string `yaml:"format,omitempty"`
}

// Archive config used for the archive
type Archive struct {
	Format          string            `yaml:"format,omitempty"`
	FormatOverrides []FormatOverride  `yaml:"format_overrides,omitempty"`
	NameTemplate    string            `yaml:"name_template,omitempty"`
	Replacements    map[string]string `yaml:"replacemnts,omitempty"`
	Files           []string          `yaml:"files,omitempty"`
}

// Release config used for the GitHub release
type Release struct {
	GitHub Repo
	Draft  bool
}

// FPM config
type FPM struct {
	Formats      []string `yaml:"formats,omitempty"`
	Dependencies []string `yaml:"dependencies,omitempty"`
	Conflicts    []string `yaml:"conflicts,omitempty"`
	Vendor       string   `yaml:"vendor,omitempty"`
	Homepage     string   `yaml:"homepage,omitempty"`
	Maintainer   string   `yaml:"maintainer,omitempty"`
	Description  string   `yaml:"description,omitempty"`
	License      string   `yaml:"license,omitempty"`
}

// Project includes all project configuration
type Project struct {
	Release Release
	Brew    Homebrew
	Build   Build
	Archive Archive
	FPM     FPM `yaml:"fpm"`

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
