package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v1"
)

var emptyBrew = HomebrewDeploy{}

type HomebrewDeploy struct {
	Repo  string
	Token string
}

type BuildConfig struct {
	Oses   []string
	Arches []string
}

type ProjectConfig struct {
	Repo       string
	Main       string
	BinaryName string
	FileList   []string
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
	return fix(config), err
}

func fix(config ProjectConfig) ProjectConfig {
	if config.BinaryName == "" {
		dir, _ := os.Getwd()
		config.BinaryName = filepath.Base(dir)
	}
	if len(config.FileList) == 0 {
		config.FileList = []string{
			"README.md",
			"LICENSE.md",
		}
	}
	if config.Main == "" {
		config.Main = "main.go"
	}
	if config.Token == "" {
		config.Token = os.Getenv("GITHUB_TOKEN")
	}
	if config.Brew != emptyBrew && config.Brew.Token == "" {
		config.Brew.Token = config.Token
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
