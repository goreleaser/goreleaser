package config

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/goreleaser/releaser/config/git"
	yaml "gopkg.in/yaml.v1"
)

var (
	emptyBrew    = Homebrew{}
	filePatterns = []string{"LICENCE*", "LICENSE*", "README*"}
)

// Homebrew contains the brew section
type Homebrew struct {
	Repo    string
	Token   string
	Caveats string
}

// BuildConfig contains the build configuration section
type BuildConfig struct {
	Oses   []string
	Arches []string
	Main   string
}

// GitInfo includes tags and diffs used in some point
type GitInfo struct {
	CurrentTag  string
	PreviousTag string
	Diff        string
}

// ProjectConfig includes all project configuration
type ProjectConfig struct {
	Repo       string
	BinaryName string `yaml:"binary_name"`
	Files      []string
	Brew       Homebrew
	Token      string
	Build      BuildConfig
	Git        GitInfo `yaml:"_"`
}

// Load config file
func Load(file string) (config ProjectConfig, err error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(data, &config)
	config = fix(config)
	config, err = fillGitData(config)
	if err != nil {
		return config, err
	}
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
		config.Files = []string{}

		for _, pattern := range filePatterns {
			matches, err := globPath(pattern)
			if err != nil {
				log.Fatalf("Error searching for %q: %v", pattern, err)
			}

			config.Files = append(config.Files, matches...)
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

func fillGitData(config ProjectConfig) (ProjectConfig, error) {
	tag, err := git.CurrentTag()
	if err != nil {
		return config, err
	}
	previous, err := git.PreviousTag(tag)
	if err != nil {
		return config, err
	}
	log, err := git.Log(previous, tag)
	if err != nil {
		return config, err
	}

	config.Git.CurrentTag = tag
	config.Git.PreviousTag = previous
	config.Git.Diff = log
	return config, nil
}

func contains(s string, ss []string) bool {
	for _, sx := range ss {
		if sx == s {
			return true
		}
	}
	return false
}

func globPath(p string) (m []string, err error) {
	var cwd string
	var dirs []string

	if cwd, err = os.Getwd(); err != nil {
		return
	}

	fp := path.Join(cwd, p)

	if dirs, err = filepath.Glob(fp); err != nil {
		return
	}

	// Normalise to avoid nested dirs in tarball
	for _, dir := range dirs {
		_, f := filepath.Split(dir)
		m = append(m, f)
	}

	return
}
