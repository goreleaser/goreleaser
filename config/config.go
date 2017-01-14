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
