// Package config contains the model and loader of the goreleaser configuration
// file.
package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/apex/log"
	yaml "gopkg.in/yaml.v2"
)

// Repo represents any kind of repo (github, gitlab, etc)
type Repo struct {
	Owner string `yaml:",omitempty"`
	Name  string `yaml:",omitempty"`

	// Capture all undefined fields and should be empty after loading
	XXX map[string]interface{} `yaml:",inline"`
}

// String of the repo, e.g. owner/name
func (r Repo) String() string {
	return r.Owner + "/" + r.Name
}

// Homebrew contains the brew section
type Homebrew struct {
	GitHub       Repo     `yaml:",omitempty"`
	Folder       string   `yaml:",omitempty"`
	Caveats      string   `yaml:",omitempty"`
	Plist        string   `yaml:",omitempty"`
	Install      string   `yaml:",omitempty"`
	Dependencies []string `yaml:",omitempty"`
	Test         string   `yaml:",omitempty"`
	Conflicts    []string `yaml:",omitempty"`
	Description  string   `yaml:",omitempty"`
	Homepage     string   `yaml:",omitempty"`

	// Capture all undefined fields and should be empty after loading
	XXX map[string]interface{} `yaml:",inline"`
}

// Hooks define actions to run before and/or after something
type Hooks struct {
	Pre  string `yaml:",omitempty"`
	Post string `yaml:",omitempty"`

	// Capture all undefined fields and should be empty after loading
	XXX map[string]interface{} `yaml:",inline"`
}

// IgnoredBuild represents a build ignored by the user
type IgnoredBuild struct {
	Goos, Goarch, Goarm string

	// Capture all undefined fields and should be empty after loading
	XXX map[string]interface{} `yaml:",inline"`
}

// Build contains the build configuration section
type Build struct {
	Goos    []string       `yaml:",omitempty"`
	Goarch  []string       `yaml:",omitempty"`
	Goarm   []string       `yaml:",omitempty"`
	Ignore  []IgnoredBuild `yaml:",omitempty"`
	Main    string         `yaml:",omitempty"`
	Ldflags string         `yaml:",omitempty"`
	Flags   string         `yaml:",omitempty"`
	Binary  string         `yaml:",omitempty"`
	Hooks   Hooks          `yaml:",omitempty"`
	Env     []string       `yaml:",omitempty"`

	// Capture all undefined fields and should be empty after loading
	XXX map[string]interface{} `yaml:",inline"`
}

// FormatOverride is used to specify a custom format for a specific GOOS.
type FormatOverride struct {
	Goos   string `yaml:",omitempty"`
	Format string `yaml:",omitempty"`

	// Capture all undefined fields and should be empty after loading
	XXX map[string]interface{} `yaml:",inline"`
}

// Archive config used for the archive
type Archive struct {
	Format          string            `yaml:",omitempty"`
	FormatOverrides []FormatOverride  `yaml:"format_overrides,omitempty"`
	NameTemplate    string            `yaml:"name_template,omitempty"`
	Replacements    map[string]string `yaml:",omitempty"`
	Files           []string          `yaml:",omitempty"`

	// Capture all undefined fields and should be empty after loading
	XXX map[string]interface{} `yaml:",inline"`
}

// Release config used for the GitHub release
type Release struct {
	GitHub Repo `yaml:",omitempty"`
	Draft  bool `yaml:",omitempty"`

	// Capture all undefined fields and should be empty after loading
	XXX map[string]interface{} `yaml:",inline"`
}

// FPM config
type FPM struct {
	Formats      []string `yaml:",omitempty"`
	Dependencies []string `yaml:",omitempty"`
	Conflicts    []string `yaml:",omitempty"`
	Vendor       string   `yaml:",omitempty"`
	Homepage     string   `yaml:",omitempty"`
	Maintainer   string   `yaml:",omitempty"`
	Description  string   `yaml:",omitempty"`
	License      string   `yaml:",omitempty"`

	// Capture all undefined fields and should be empty after loading
	XXX map[string]interface{} `yaml:",inline"`
}

// AppsMetadata for the binaries that will be in the snap package
type SnapcraftAppMetadata struct {
	Plugs  []string
	Daemon string
}

// Snapcraft config
type Snapcraft struct {
	Summary     string                          `yaml:",omitempty"`
	Description string                          `yaml:",omitempty"`
	Grade       string                          `yaml:",omitempty"`
	Confinement string                          `yaml:",omitempty"`
	Apps        map[string]SnapcraftAppMetadata `yaml:",omitempty"`

	// Capture all undefined fields and should be empty after loading
	XXX map[string]interface{} `yaml:",inline"`
}

// Snapshot config
type Snapshot struct {
	NameTemplate string `yaml:"name_template,omitempty"`

	// Capture all undefined fields and should be empty after loading
	XXX map[string]interface{} `yaml:",inline"`
}

// Project includes all project configuration
type Project struct {
	ProjectName string    `yaml:"project_name,omitempty"`
	Release     Release   `yaml:",omitempty"`
	Brew        Homebrew  `yaml:",omitempty"`
	Builds      []Build   `yaml:",omitempty"`
	Archive     Archive   `yaml:",omitempty"`
	FPM         FPM       `yaml:",omitempty"`
	Snapcraft   Snapcraft `yaml:",omitempty"`
	Snapshot    Snapshot  `yaml:",omitempty"`

	// this is a hack ¯\_(ツ)_/¯
	SingleBuild Build `yaml:"build,omitempty"`

	// test only property indicating the path to the dist folder
	Dist string `yaml:"-"`

	// Capture all undefined fields and should be empty after loading
	XXX map[string]interface{} `yaml:",inline"`
}

// Load config file
func Load(file string) (config Project, err error) {
	f, err := os.Open(file)
	if err != nil {
		return
	}
	log.WithField("file", file).Info("loading config file")
	return LoadReader(f)
}

// LoadReader config via io.Reader
func LoadReader(fd io.Reader) (config Project, err error) {
	data, err := ioutil.ReadAll(fd)
	if err != nil {
		return config, err
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, err
	}
	log.WithField("config", config).Debug("loaded config file")
	return config, checkOverflows(config)
}

func checkOverflows(config Project) error {
	var checker = &overflowChecker{}
	checker.check(config.XXX, "")
	checker.check(config.Archive.XXX, "archive")
	for i, ov := range config.Archive.FormatOverrides {
		checker.check(ov.XXX, fmt.Sprintf("archive.format_overrides[%d]", i))
	}
	checker.check(config.Brew.XXX, "brew")
	checker.check(config.Brew.GitHub.XXX, "brew.github")
	for i, build := range config.Builds {
		checker.check(build.XXX, fmt.Sprintf("builds[%d]", i))
		checker.check(build.Hooks.XXX, fmt.Sprintf("builds[%d].hooks", i))
		for j, ignored := range build.Ignore {
			checker.check(ignored.XXX, fmt.Sprintf("builds[%d].ignored_builds[%d]", i, j))
		}
	}
	checker.check(config.FPM.XXX, "fpm")
	checker.check(config.Snapcraft.XXX, "snapcraft")
	checker.check(config.Release.XXX, "release")
	checker.check(config.Release.GitHub.XXX, "release.github")
	checker.check(config.SingleBuild.XXX, "build")
	checker.check(config.SingleBuild.Hooks.XXX, "builds.hooks")
	for i, ignored := range config.SingleBuild.Ignore {
		checker.check(ignored.XXX, fmt.Sprintf("builds.ignored_builds[%d]", i))
	}
	checker.check(config.Snapshot.XXX, "snapshot")
	return checker.err()
}

type overflowChecker struct {
	fields []string
}

func (o *overflowChecker) check(m map[string]interface{}, ctx string) {
	for k := range m {
		var key = fmt.Sprintf("%s.%s", ctx, k)
		if ctx == "" {
			key = k
		}
		o.fields = append(o.fields, key)
	}
}

func (o *overflowChecker) err() error {
	if len(o.fields) == 0 {
		return nil
	}
	return fmt.Errorf(
		"unknown fields in the config file: %s",
		strings.Join(o.fields, ", "),
	)
}
