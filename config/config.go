package config

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v1"
)

// Homebrew contains the brew section
type Homebrew struct {
	Repo    string
	Folder  string
	Caveats string
}

// BuildConfig contains the build configuration section
type BuildConfig struct {
	Oses    []string
	Arches  []string
	Main    string
	Ldflags string
}

// ArchiveConfig config used for the archive
type ArchiveConfig struct {
	Format       string
	NameTemplate string `yaml:"name_template"`
	Replacements map[string]string
}

// ProjectConfig includes all project configuration
type ProjectConfig struct {
	Repo       string
	BinaryName string `yaml:"binary_name"`
	Files      []string
	Brew       Homebrew
	Build      BuildConfig
	Archive    ArchiveConfig
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
