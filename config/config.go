// Package config contains the model and loader of the goreleaser configuration
// file.
package config

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v1"
)

// Repo represents any kind of repo (github, gitlab, etc)
type Repo struct {
	Owner string
	Name  string
}

// String of the repo, e.g. owner/name
func (r Repo) String() string {
	return r.Owner + "/" + r.Name
}

// Homebrew contains the brew section
type Homebrew struct {
	Repo         string // deprecated!
	GitHub       Repo
	Folder       string
	Caveats      string
	Plist        string
	Install      string
	Dependencies []string
	Conflicts    []string
}

// Hooks define actions to run before and/or after something
type Hooks struct {
	Pre  string
	Post string
}

// Build contains the build configuration section
type Build struct {
	Goos       []string
	Goarch     []string
	Main       string
	Ldflags    string
	Flags      string
	BinaryName string `yaml:"binary_name"` // deprecated
	Binary     string
	Hooks      Hooks
}

// Archive config used for the archive
type Archive struct {
	Format       string
	NameTemplate string `yaml:"name_template"`
	Replacements map[string]string
	Files        []string
}

// Release config used for the GitHub release
type Release struct {
	Repo   string // deprecated!
	GitHub Repo
	Draft  bool
}

// FPM config
type FPM struct {
	Formats      []string
	Dependencies []string
	Conflicts    []string
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
