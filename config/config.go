package config

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v1"
)

var emptyBrew = HomebrewDeploy{}

type HomebrewDeploy struct {
	Repo  string
	Token string
}

type ProjectConfig struct {
	Repo       string
	Main       string
	BinaryName string
	FileList   []string
	Brew       HomebrewDeploy
	Token      string
}

func Load(file string) (config ProjectConfig, err error) {
	log.Println("Loading", file)
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
			config.BinaryName,
		}
	} else {
		if !contains(config.BinaryName, config.FileList) {
			config.FileList = append(config.FileList, config.BinaryName)
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
	return config
}

func contains(s string, ss []string) (ok bool) {
	for _, sx := range ss {
		if sx == s {
			ok = true
			break
		}
	}
	return ok
}
