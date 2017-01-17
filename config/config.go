package config

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v1"
)

// Homebrew contains the brew section
type Homebrew struct {
	Repo         string
	Folder       string
	Caveats      string
	Dependencies []string
}

// Build contains the build configuration section
type Build struct {
	Goos       []string
	Goarch     []string
	Main       string
	Ldflags    string
	BinaryName string `yaml:"binary_name"`
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
	Repo string
}

// Project includes all project configuration
type Project struct {
	Release Release
	Brew    Homebrew
	Build   Build
	Archive Archive
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
