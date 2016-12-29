package config

import (
	"errors"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v1"
)

var emptyBrew = HomebrewDeploy{}

type HomebrewDeploy struct {
	Repo    string
	Token   string
	Caveats string
}

type BuildConfig struct {
	Oses   []string
	Arches []string
	Main   string
}

type ProjectConfig struct {
	Repo       string
	BinaryName string `yaml:"binary_name"`
	Files      []string
	Brew       HomebrewDeploy
	Token      string
	Build      BuildConfig
}

func Load(file string) (config ProjectConfig, err error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(data, &config)
	config = fix(config)
	if config.BinaryName == "" {
		return config, errors.New("missing binary_name")
	}
	if config.Repo == "" {
		return config, errors.New("missing repo")
	}
	return config, err
}

func fix(config ProjectConfig) ProjectConfig {
	if len(config.Files) == 0 {
		config.Files = []string{
			"README.md",
			"LICENSE.md",
		}
	}
	if config.Token == "" {
		config.Token = os.Getenv("GITHUB_TOKEN")
	}
	if config.Brew != emptyBrew && config.Brew.Token == "" {
		config.Brew.Token = config.Token
	}
	if config.Build.Main == "" {
		config.Build.Main = "main.go"
	}
	if len(config.Build.Oses) == 0 {
		config.Build.Oses = []string{"linux", "darwin"}
	}
	if len(config.Build.Arches) == 0 {
		config.Build.Arches = []string{"amd64", "386"}
	}
	return config
}

func contains(s string, ss []string) bool {
	for _, sx := range ss {
		if sx == s {
			return true
		}
	}
	return false
}
