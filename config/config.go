package config

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/goreleaser/releaser/config/git"
	yaml "gopkg.in/yaml.v1"
)

var filePatterns = []string{"LICENCE*", "LICENSE*", "README*", "CHANGELOG*"}

// Homebrew contains the brew section
type Homebrew struct {
	Repo    string
	Token   string `yaml:"-"`
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

// ArchiveConfig
type ArchiveConfig struct {
	Format string
}

// ProjectConfig includes all project configuration
type ProjectConfig struct {
	Repo         string
	BinaryName   string `yaml:"binary_name"`
	Files        []string
	Brew         Homebrew
	Token        string `yaml:"-"`
	Build        BuildConfig
	Git          GitInfo `yaml:"-"`
	NameTemplate string  `yaml:"name_template"`
	Archive      ArchiveConfig
}

// Load config file
func Load(file string) (config ProjectConfig, err error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return config, err
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, err
	}
	config.fillBasicData()
	if err := config.fillFiles(); err != nil {
		return config, err
	}
	if err := config.fillGitData(); err != nil {
		return config, err
	}
	return config, config.validate()
}

func (config *ProjectConfig) validate() (err error) {
	if config.BinaryName == "" {
		return errors.New("missing binary_name")
	}
	if config.Repo == "" {
		return errors.New("missing repo")
	}
	return
}

func (config *ProjectConfig) fillFiles() (err error) {
	if len(config.Files) != 0 {
		return
	}
	config.Files = []string{}
	for _, pattern := range filePatterns {
		matches, err := globPath(pattern)
		if err != nil {
			return err
		}

		config.Files = append(config.Files, matches...)
	}
	return
}

func (config *ProjectConfig) fillBasicData() {
	if config.Token == "" {
		config.Token = os.Getenv("GITHUB_TOKEN")
	}
	if config.Brew.Repo != "" {
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
	if config.NameTemplate == "" {
		config.NameTemplate = "{{.BinaryName}}_{{.Os}}_{{.Arch}}"
	}
	if config.Archive.Format == "" {
		config.Archive.Format = "tar.gz"
	}
}

func (config *ProjectConfig) fillGitData() (err error) {
	tag, err := git.CurrentTag()
	if err != nil {
		return
	}
	previous, err := git.PreviousTag(tag)
	if err != nil {
		return
	}
	log, err := git.Log(previous, tag)
	if err != nil {
		return
	}

	config.Git.CurrentTag = tag
	config.Git.PreviousTag = previous
	config.Git.Diff = log
	return
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
