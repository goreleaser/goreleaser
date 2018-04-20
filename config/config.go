// Package config contains the model and loader of the goreleaser configuration
// file.
package config

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/apex/log"
	"gopkg.in/yaml.v2"
)

// GitHubURLs holds the URLs to be used when using github enterprise
type GitHubURLs struct {
	API      string `yaml:"api,omitempty"`
	Upload   string `yaml:"upload,omitempty"`
	Download string `yaml:"download,omitempty"`
}

// Repo represents any kind of repo (github, gitlab, etc)
type Repo struct {
	Owner string `yaml:",omitempty"`
	Name  string `yaml:",omitempty"`
}

// String of the repo, e.g. owner/name
func (r Repo) String() string {
	if r.Owner == "" && r.Name == "" {
		return ""
	}
	return r.Owner + "/" + r.Name
}

// Homebrew contains the brew section
type Homebrew struct {
	Name              string       `yaml:",omitempty"`
	GitHub            Repo         `yaml:",omitempty"`
	CommitAuthor      CommitAuthor `yaml:"commit_author,omitempty"`
	Folder            string       `yaml:",omitempty"`
	Caveats           string       `yaml:",omitempty"`
	Plist             string       `yaml:",omitempty"`
	Install           string       `yaml:",omitempty"`
	Dependencies      []string     `yaml:",omitempty"`
	BuildDependencies []string     `yaml:"build_dependencies,omitempty"`
	Test              string       `yaml:",omitempty"`
	Conflicts         []string     `yaml:",omitempty"`
	Description       string       `yaml:",omitempty"`
	Homepage          string       `yaml:",omitempty"`
	SkipUpload        bool         `yaml:"skip_upload,omitempty"`
	DownloadStrategy  string       `yaml:"download_strategy,omitempty"`
	SourceTarball     string       `yaml:"-"`
}

// Scoop contains the scoop.sh section
type Scoop struct {
	Bucket       Repo         `yaml:",omitempty"`
	CommitAuthor CommitAuthor `yaml:"commit_author,omitempty"`
	Homepage     string       `yaml:",omitempty"`
	Description  string       `yaml:",omitempty"`
	License      string       `yaml:",omitempty"`
}

// CommitAuthor is the author of a Git commit
type CommitAuthor struct {
	Name  string `yaml:",omitempty"`
	Email string `yaml:",omitempty"`
}

// Hooks define actions to run before and/or after something
type Hooks struct {
	Pre  string `yaml:",omitempty"`
	Post string `yaml:",omitempty"`
}

// IgnoredBuild represents a build ignored by the user
type IgnoredBuild struct {
	Goos, Goarch, Goarm string
}

// Build contains the build configuration section
type Build struct {
	Goos     []string       `yaml:",omitempty"`
	Goarch   []string       `yaml:",omitempty"`
	Goarm    []string       `yaml:",omitempty"`
	Targets  []string       `yaml:",omitempty"`
	Ignore   []IgnoredBuild `yaml:",omitempty"`
	Main     string         `yaml:",omitempty"`
	Ldflags  string         `yaml:",omitempty"`
	Flags    string         `yaml:",omitempty"`
	Binary   string         `yaml:",omitempty"`
	Hooks    Hooks          `yaml:",omitempty"`
	Env      []string       `yaml:",omitempty"`
	Lang     string         `yaml:",omitempty"`
	Asmflags string         `yaml:",omitempty"`
	Gcflags  string         `yaml:",omitempty"`
}

// FormatOverride is used to specify a custom format for a specific GOOS.
type FormatOverride struct {
	Goos   string `yaml:",omitempty"`
	Format string `yaml:",omitempty"`
}

// Archive config used for the archive
type Archive struct {
	NameTemplate string            `yaml:"name_template,omitempty"`
	Replacements map[string]string `yaml:",omitempty"`

	Format          string           `yaml:",omitempty"`
	FormatOverrides []FormatOverride `yaml:"format_overrides,omitempty"`
	WrapInDirectory bool             `yaml:"wrap_in_directory,omitempty"`
	Files           []string         `yaml:",omitempty"`
}

// Release config used for the GitHub release
type Release struct {
	GitHub       Repo   `yaml:",omitempty"`
	Draft        bool   `yaml:",omitempty"`
	Prerelease   bool   `yaml:",omitempty"`
	NameTemplate string `yaml:"name_template,omitempty"`
}

// NFPM config
type NFPM struct {
	NameTemplate string            `yaml:"name_template,omitempty"`
	Replacements map[string]string `yaml:",omitempty"`

	Formats      []string          `yaml:",omitempty"`
	Dependencies []string          `yaml:",omitempty"`
	Recommends   []string          `yaml:",omitempty"`
	Suggests     []string          `yaml:",omitempty"`
	Conflicts    []string          `yaml:",omitempty"`
	Vendor       string            `yaml:",omitempty"`
	Homepage     string            `yaml:",omitempty"`
	Maintainer   string            `yaml:",omitempty"`
	Description  string            `yaml:",omitempty"`
	License      string            `yaml:",omitempty"`
	Bindir       string            `yaml:",omitempty"`
	Files        map[string]string `yaml:",omitempty"`
	ConfigFiles  map[string]string `yaml:"config_files,omitempty"`
	Scripts      NFPMScripts       `yaml:"scripts,omitempty"`
}

// NFPMScripts is used to specify maintainer scripts
type NFPMScripts struct {
	PreInstall  string `yaml:"preinstall,omitempty"`
	PostInstall string `yaml:"postinstall,omitempty"`
	PreRemove   string `yaml:"preremove,omitempty"`
	PostRemove  string `yaml:"postremove,omitempty"`
}

// Sign config
type Sign struct {
	Cmd       string   `yaml:"cmd,omitempty"`
	Args      []string `yaml:"args,omitempty"`
	Signature string   `yaml:"signature,omitempty"`
	Artifacts string   `yaml:"artifacts,omitempty"`
}

// SnapcraftAppMetadata for the binaries that will be in the snap package
type SnapcraftAppMetadata struct {
	Plugs  []string
	Daemon string
}

// Snapcraft config
type Snapcraft struct {
	NameTemplate string            `yaml:"name_template,omitempty"`
	Replacements map[string]string `yaml:",omitempty"`

	Name        string                          `yaml:",omitempty"`
	Summary     string                          `yaml:",omitempty"`
	Description string                          `yaml:",omitempty"`
	Grade       string                          `yaml:",omitempty"`
	Confinement string                          `yaml:",omitempty"`
	Apps        map[string]SnapcraftAppMetadata `yaml:",omitempty"`
}

// Snapshot config
type Snapshot struct {
	NameTemplate string `yaml:"name_template,omitempty"`
}

// Checksum config
type Checksum struct {
	NameTemplate string `yaml:"name_template,omitempty"`
}

// Docker image config
type Docker struct {
	Binary         string   `yaml:",omitempty"`
	Goos           string   `yaml:",omitempty"`
	Goarch         string   `yaml:",omitempty"`
	Goarm          string   `yaml:",omitempty"`
	Image          string   `yaml:",omitempty"`
	Dockerfile     string   `yaml:",omitempty"`
	Latest         bool     `yaml:",omitempty"`
	SkipPush       bool     `yaml:"skip_push,omitempty"`
	OldTagTemplate string   `yaml:"tag_template,omitempty"`
	TagTemplates   []string `yaml:"tag_templates,omitempty"`
	Files          []string `yaml:"extra_files,omitempty"`
}

// Artifactory server configuration
type Artifactory struct {
	Target   string `yaml:",omitempty"`
	Name     string `yaml:",omitempty"`
	Username string `yaml:",omitempty"`
	Mode     string `yaml:",omitempty"`
}

// Filters config
type Filters struct {
	Exclude []string `yaml:",omitempty"`
}

// Changelog Config
type Changelog struct {
	Filters Filters `yaml:",omitempty"`
	Sort    string  `yaml:",omitempty"`
}

// EnvFiles holds paths to files that contains environment variables
// values like the github token for example
type EnvFiles struct {
	GitHubToken string `yaml:"github_token,omitempty"`
}

// Git config
type Git struct {
	ShortHash bool `yaml:"short_hash,omitempty"`
}

// Before config
type Before struct {
	Hooks []string `yaml:",omitempty"`
}

// Project includes all project configuration
type Project struct {
	ProjectName   string        `yaml:"project_name,omitempty"`
	Release       Release       `yaml:",omitempty"`
	Brew          Homebrew      `yaml:",omitempty"`
	Scoop         Scoop         `yaml:",omitempty"`
	Builds        []Build       `yaml:",omitempty"`
	Archive       Archive       `yaml:",omitempty"`
	FPM           NFPM          `yaml:",omitempty"` // deprecated
	NFPM          NFPM          `yaml:",omitempty"`
	Snapcraft     Snapcraft     `yaml:",omitempty"`
	Snapshot      Snapshot      `yaml:",omitempty"`
	Checksum      Checksum      `yaml:",omitempty"`
	Dockers       []Docker      `yaml:",omitempty"`
	Artifactories []Artifactory `yaml:",omitempty"`
	Changelog     Changelog     `yaml:",omitempty"`
	Dist          string        `yaml:",omitempty"`
	Sign          Sign          `yaml:",omitempty"`
	EnvFiles      EnvFiles      `yaml:"env_files,omitempty"`
	Git           Git           `yaml:",omitempty"`
	Before        Before        `yaml:",omitempty"`

	// this is a hack ¯\_(ツ)_/¯
	SingleBuild Build `yaml:"build,omitempty"`

	// should be set if using github enterprise
	GitHubURLs GitHubURLs `yaml:"github_urls,omitempty"`
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
	err = yaml.UnmarshalStrict(data, &config)
	log.WithField("config", config).Debug("loaded config file")
	return config, err
}
