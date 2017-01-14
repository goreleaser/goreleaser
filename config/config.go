package config

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v1"
)

// HomebrewConfig contains the brew section
type HomebrewConfig struct {
	Repo    string
	Folder  string
	Caveats string
}

// BuildConfig contains the build configuration section
type BuildConfig struct {
	Goos       []string
	Goarch     []string
	Main       string
	Ldflags    string
	BinaryName string `yaml:"binary_name"`
}

// ArchiveConfig config used for the archive
type ArchiveConfig struct {
	Format       string
	NameTemplate string `yaml:"name_template"`
	Replacements map[string]string
	Files        []string
}

// ReleaseConfig config used for the GitHub release
type ReleaseConfig struct {
	Repo string
}

// ProjectConfig includes all project configuration
type ProjectConfig struct {
	Release ReleaseConfig
	Brew    HomebrewConfig
	Build   BuildConfig
	Archive ArchiveConfig
}

// Load config file
func Load(file string) (config ProjectConfig, err error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(data, &config)
	return
}
