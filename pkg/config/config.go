// Package config contains the model and loader of the goreleaser configuration
// file.
package config

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/goreleaser/nfpm/v2/files"
	"github.com/invopop/jsonschema"
)

type Versioned struct {
	Version int
}

// Git configs.
type Git struct {
	TagSort          string   `yaml:"tag_sort,omitempty" json:"tag_sort,omitempty" jsonschema:"enum=-version:refname,enum=-version:creatordate,default=-version:refname"`
	PrereleaseSuffix string   `yaml:"prerelease_suffix,omitempty" json:"prerelease_suffix,omitempty"`
	IgnoreTags       []string `yaml:"ignore_tags,omitempty" json:"ignore_tags,omitempty"`
}

// GitHubURLs holds the URLs to be used when using github enterprise.
type GitHubURLs struct {
	API           string `yaml:"api,omitempty" json:"api,omitempty"`
	Upload        string `yaml:"upload,omitempty" json:"upload,omitempty"`
	Download      string `yaml:"download,omitempty" json:"download,omitempty"`
	SkipTLSVerify bool   `yaml:"skip_tls_verify,omitempty" json:"skip_tls_verify,omitempty"`
}

// GitLabURLs holds the URLs to be used when using gitlab ce/enterprise.
type GitLabURLs struct {
	API                string `yaml:"api,omitempty" json:"api,omitempty"`
	Download           string `yaml:"download,omitempty" json:"download,omitempty"`
	SkipTLSVerify      bool   `yaml:"skip_tls_verify,omitempty" json:"skip_tls_verify,omitempty"`
	UsePackageRegistry bool   `yaml:"use_package_registry,omitempty" json:"use_package_registry,omitempty"`
	UseJobToken        bool   `yaml:"use_job_token,omitempty" json:"use_job_token,omitempty"`
}

// GiteaURLs holds the URLs to be used when using gitea.
type GiteaURLs struct {
	API           string `yaml:"api,omitempty" json:"api,omitempty"`
	Download      string `yaml:"download,omitempty" json:"download,omitempty"`
	SkipTLSVerify bool   `yaml:"skip_tls_verify,omitempty" json:"skip_tls_verify,omitempty"`
}

// Repo represents any kind of repo (github, gitlab, etc).
// to upload releases into.
type Repo struct {
	Owner  string `yaml:"owner,omitempty" json:"owner,omitempty"`
	Name   string `yaml:"name,omitempty" json:"name,omitempty"`
	RawURL string `yaml:"-" json:"-"`
}

// String of the repo, e.g. owner/name.
func (r Repo) String() string {
	if r.isSCM() {
		return r.Owner + "/" + r.Name
	}
	return r.Owner
}

// CheckSCM returns an error if the given url is not a valid scm url.
func (r Repo) CheckSCM() error {
	if r.isSCM() {
		return nil
	}
	return fmt.Errorf("invalid scm url: %s", r.RawURL)
}

// isSCM returns true if the repo has both an owner and name.
func (r Repo) isSCM() bool {
	return r.Owner != "" && r.Name != ""
}

// RepoRef represents any kind of repo which may differ
// from the one we are building from and may therefore
// also require separate authentication
// e.g. Homebrew Tap, Scoop bucket.
type RepoRef struct {
	Owner  string `yaml:"owner,omitempty" json:"owner,omitempty"`
	Name   string `yaml:"name,omitempty" json:"name,omitempty"`
	Token  string `yaml:"token,omitempty" json:"token,omitempty"`
	Branch string `yaml:"branch,omitempty" json:"branch,omitempty"`

	Git         GitRepoRef  `yaml:"git,omitempty" json:"git,omitempty"`
	PullRequest PullRequest `yaml:"pull_request,omitempty" json:"pull_request,omitempty"`
}

type GitRepoRef struct {
	URL        string `yaml:"url,omitempty" json:"url,omitempty"`
	SSHCommand string `yaml:"ssh_command,omitempty" json:"ssh_command,omitempty"`
	PrivateKey string `yaml:"private_key,omitempty" json:"private_key,omitempty"`
}

type PullRequestBase struct {
	Owner  string `yaml:"owner,omitempty" json:"owner,omitempty"`
	Name   string `yaml:"name,omitempty" json:"name,omitempty"`
	Branch string `yaml:"branch,omitempty" json:"branch,omitempty"`
}

// type alias to prevent stack overflowing in the custom unmarshaler.
type pullRequestBase PullRequestBase

// UnmarshalYAML is a custom unmarshaler that accept brew deps in both the old and new format.
func (a *PullRequestBase) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err == nil {
		a.Branch = str
		return nil
	}

	var base pullRequestBase
	if err := unmarshal(&base); err != nil {
		return err
	}

	a.Branch = base.Branch
	a.Owner = base.Owner
	a.Name = base.Name

	return nil
}

func (a PullRequestBase) JSONSchema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	schema := reflector.Reflect(&pullRequestBase{})
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{
				Type: "string",
			},
			schema,
		},
	}
}

type PullRequest struct {
	Enabled bool            `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Base    PullRequestBase `yaml:"base,omitempty" json:"base,omitempty"`
	Draft   bool            `yaml:"draft,omitempty" json:"draft,omitempty"`
}

// HomebrewDependency represents Homebrew dependency.
type HomebrewDependency struct {
	Name    string `yaml:"name,omitempty" json:"name,omitempty"`
	Type    string `yaml:"type,omitempty" json:"type,omitempty"`
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
	OS      string `yaml:"os,omitempty" json:"os,omitempty" jsonschema:"enum=mac,enum=linux"`
}

// type alias to prevent stack overflowing in the custom unmarshaler.
type homebrewDependency HomebrewDependency

// UnmarshalYAML is a custom unmarshaler that accept brew deps in both the old and new format.
func (a *HomebrewDependency) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err == nil {
		a.Name = str
		return nil
	}

	var dep homebrewDependency
	if err := unmarshal(&dep); err != nil {
		return err
	}

	a.Name = dep.Name
	a.Type = dep.Type

	return nil
}

func (a HomebrewDependency) JSONSchema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	schema := reflector.Reflect(&homebrewDependency{})
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{
				Type: "string",
			},
			schema,
		},
	}
}

type AUR struct {
	Name                  string       `yaml:"name,omitempty" json:"name,omitempty"`
	IDs                   []string     `yaml:"ids,omitempty" json:"ids,omitempty"`
	CommitAuthor          CommitAuthor `yaml:"commit_author,omitempty" json:"commit_author,omitempty"`
	CommitMessageTemplate string       `yaml:"commit_msg_template,omitempty" json:"commit_msg_template,omitempty"`
	Description           string       `yaml:"description,omitempty" json:"description,omitempty"`
	Homepage              string       `yaml:"homepage,omitempty" json:"homepage,omitempty"`
	License               string       `yaml:"license,omitempty" json:"license,omitempty"`
	SkipUpload            string       `yaml:"skip_upload,omitempty" json:"skip_upload,omitempty" jsonschema:"oneof_type=string;boolean"`
	URLTemplate           string       `yaml:"url_template,omitempty" json:"url_template,omitempty"`
	Maintainers           []string     `yaml:"maintainers,omitempty" json:"maintainers,omitempty"`
	Contributors          []string     `yaml:"contributors,omitempty" json:"contributors,omitempty"`
	Provides              []string     `yaml:"provides,omitempty" json:"provides,omitempty"`
	Conflicts             []string     `yaml:"conflicts,omitempty" json:"conflicts,omitempty"`
	Depends               []string     `yaml:"depends,omitempty" json:"depends,omitempty"`
	OptDepends            []string     `yaml:"optdepends,omitempty" json:"optdepends,omitempty"`
	Backup                []string     `yaml:"backup,omitempty" json:"backup,omitempty"`
	Rel                   string       `yaml:"rel,omitempty" json:"rel,omitempty"`
	Package               string       `yaml:"package,omitempty" json:"package,omitempty"`
	GitURL                string       `yaml:"git_url,omitempty" json:"git_url,omitempty"`
	GitSSHCommand         string       `yaml:"git_ssh_command,omitempty" json:"git_ssh_command,omitempty"`
	PrivateKey            string       `yaml:"private_key,omitempty" json:"private_key,omitempty"`
	Goamd64               string       `yaml:"goamd64,omitempty" json:"goamd64,omitempty"`
	Directory             string       `yaml:"directory,omitempty" json:"directory,omitempty"`
}

// Homebrew contains the brew section.
type Homebrew struct {
	Name                  string               `yaml:"name,omitempty" json:"name,omitempty"`
	Repository            RepoRef              `yaml:"repository,omitempty" json:"repository,omitempty"`
	CommitAuthor          CommitAuthor         `yaml:"commit_author,omitempty" json:"commit_author,omitempty"`
	CommitMessageTemplate string               `yaml:"commit_msg_template,omitempty" json:"commit_msg_template,omitempty"`
	Directory             string               `yaml:"directory,omitempty" json:"directory,omitempty"`
	Caveats               string               `yaml:"caveats,omitempty" json:"caveats,omitempty"`
	Install               string               `yaml:"install,omitempty" json:"install,omitempty"`
	ExtraInstall          string               `yaml:"extra_install,omitempty" json:"extra_install,omitempty"`
	PostInstall           string               `yaml:"post_install,omitempty" json:"post_install,omitempty"`
	Dependencies          []HomebrewDependency `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Test                  string               `yaml:"test,omitempty" json:"test,omitempty"`
	Conflicts             []string             `yaml:"conflicts,omitempty" json:"conflicts,omitempty"`
	Description           string               `yaml:"description,omitempty" json:"description,omitempty"`
	Homepage              string               `yaml:"homepage,omitempty" json:"homepage,omitempty"`
	License               string               `yaml:"license,omitempty" json:"license,omitempty"`
	SkipUpload            string               `yaml:"skip_upload,omitempty" json:"skip_upload,omitempty" jsonschema:"oneof_type=string;boolean"`
	DownloadStrategy      string               `yaml:"download_strategy,omitempty" json:"download_strategy,omitempty"`
	URLTemplate           string               `yaml:"url_template,omitempty" json:"url_template,omitempty"`
	URLHeaders            []string             `yaml:"url_headers,omitempty" json:"url_headers,omitempty"`
	CustomRequire         string               `yaml:"custom_require,omitempty" json:"custom_require,omitempty"`
	CustomBlock           string               `yaml:"custom_block,omitempty" json:"custom_block,omitempty"`
	IDs                   []string             `yaml:"ids,omitempty" json:"ids,omitempty"`
	Goarm                 string               `yaml:"goarm,omitempty" json:"goarm,omitempty" jsonschema:"oneof_type=string;integer"`
	Goamd64               string               `yaml:"goamd64,omitempty" json:"goamd64,omitempty"`
	Service               string               `yaml:"service,omitempty" json:"service,omitempty"`

	// Deprecated: use Repository instead.
	Tap RepoRef `yaml:"tap,omitempty" json:"tap,omitempty" jsonschema:"deprecated=true,description=use repository instead"`

	// Deprecated: use Service instead.
	Plist string `yaml:"plist,omitempty" json:"plist,omitempty" jsonschema:"deprecated=true,description=use service instead"`

	// Deprecated: use Directory instead.
	Folder string `yaml:"folder,omitempty" json:"folder,omitempty" jsonschema:"deprecated=true"`
}

type Nix struct {
	Name                  string       `yaml:"name,omitempty" json:"name,omitempty"`
	Path                  string       `yaml:"path,omitempty" json:"path,omitempty"`
	Repository            RepoRef      `yaml:"repository,omitempty" json:"repository,omitempty"`
	CommitAuthor          CommitAuthor `yaml:"commit_author,omitempty" json:"commit_author,omitempty"`
	CommitMessageTemplate string       `yaml:"commit_msg_template,omitempty" json:"commit_msg_template,omitempty"`
	IDs                   []string     `yaml:"ids,omitempty" json:"ids,omitempty"`
	Goamd64               string       `yaml:"goamd64,omitempty" json:"goamd64,omitempty"`
	SkipUpload            string       `yaml:"skip_upload,omitempty" json:"skip_upload,omitempty" jsonschema:"oneof_type=string;boolean"`
	URLTemplate           string       `yaml:"url_template,omitempty" json:"url_template,omitempty"`
	Install               string       `yaml:"install,omitempty" json:"install,omitempty"`
	ExtraInstall          string       `yaml:"extra_install,omitempty" json:"extra_install,omitempty"`
	PostInstall           string       `yaml:"post_install,omitempty" json:"post_install,omitempty"`
	Description           string       `yaml:"description,omitempty" json:"description,omitempty"`
	Homepage              string       `yaml:"homepage,omitempty" json:"homepage,omitempty"`
	License               string       `yaml:"license,omitempty" json:"license,omitempty"`

	Dependencies []NixDependency `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
}

type NixDependency struct {
	Name string `yaml:"name" json:"name"`
	OS   string `yaml:"os,omitempty" json:"os,omitempty" jsonschema:"enum=linux,enum=darwin"`
}

func (a NixDependency) JSONSchema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	type nixDependencyAlias NixDependency
	schema := reflector.Reflect(&nixDependencyAlias{})
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{
				Type: "string",
			},
			schema,
		},
	}
}

func (a *NixDependency) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err == nil {
		a.Name = str
		return nil
	}

	type t NixDependency
	var dep t
	if err := unmarshal(&dep); err != nil {
		return err
	}

	a.Name = dep.Name
	a.OS = dep.OS

	return nil
}

type Winget struct {
	Name                  string             `yaml:"name,omitempty" json:"name,omitempty"`
	PackageIdentifier     string             `yaml:"package_identifier,omitempty" json:"package_identifier,omitempty"`
	Publisher             string             `yaml:"publisher" json:"publisher"`
	PublisherURL          string             `yaml:"publisher_url,omitempty" json:"publisher_url,omitempty"`
	PublisherSupportURL   string             `yaml:"publisher_support_url,omitempty" json:"publisher_support_url,omitempty"`
	Copyright             string             `yaml:"copyright,omitempty" json:"copyright,omitempty"`
	CopyrightURL          string             `yaml:"copyright_url,omitempty" json:"copyright_url,omitempty"`
	Author                string             `yaml:"author,omitempty" json:"author,omitempty"`
	Path                  string             `yaml:"path,omitempty" json:"path,omitempty"`
	Repository            RepoRef            `yaml:"repository" json:"repository"`
	CommitAuthor          CommitAuthor       `yaml:"commit_author,omitempty" json:"commit_author,omitempty"`
	CommitMessageTemplate string             `yaml:"commit_msg_template,omitempty" json:"commit_msg_template,omitempty"`
	IDs                   []string           `yaml:"ids,omitempty" json:"ids,omitempty"`
	Goamd64               string             `yaml:"goamd64,omitempty" json:"goamd64,omitempty"`
	SkipUpload            string             `yaml:"skip_upload,omitempty" json:"skip_upload,omitempty" jsonschema:"oneof_type=string;boolean"`
	URLTemplate           string             `yaml:"url_template,omitempty" json:"url_template,omitempty"`
	ShortDescription      string             `yaml:"short_description" json:"short_description"`
	Description           string             `yaml:"description,omitempty" json:"description,omitempty"`
	Homepage              string             `yaml:"homepage,omitempty" json:"homepage,omitempty"`
	License               string             `yaml:"license" json:"license"`
	LicenseURL            string             `yaml:"license_url,omitempty" json:"license_url,omitempty"`
	ReleaseNotes          string             `yaml:"release_notes,omitempty" json:"release_notes,omitempty"`
	ReleaseNotesURL       string             `yaml:"release_notes_url,omitempty" json:"release_notes_url,omitempty"`
	Tags                  []string           `yaml:"tags,omitempty" json:"tags,omitempty"`
	Dependencies          []WingetDependency `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
}

type WingetDependency struct {
	PackageIdentifier string `yaml:"package_identifier" json:"package_identifier"`
	MinimumVersion    string `yaml:"minimum_version,omitempty" json:"minimum_version,omitempty"`
}

// Krew contains the krew section.
type Krew struct {
	IDs                   []string     `yaml:"ids,omitempty" json:"ids,omitempty"`
	Name                  string       `yaml:"name,omitempty" json:"name,omitempty"`
	Repository            RepoRef      `yaml:"repository,omitempty" json:"repository,omitempty"`
	CommitAuthor          CommitAuthor `yaml:"commit_author,omitempty" json:"commit_author,omitempty"`
	CommitMessageTemplate string       `yaml:"commit_msg_template,omitempty" json:"commit_msg_template,omitempty"`
	Caveats               string       `yaml:"caveats,omitempty" json:"caveats,omitempty"`
	ShortDescription      string       `yaml:"short_description,omitempty" json:"short_description,omitempty"`
	Description           string       `yaml:"description,omitempty" json:"description,omitempty"`
	Homepage              string       `yaml:"homepage,omitempty" json:"homepage,omitempty"`
	URLTemplate           string       `yaml:"url_template,omitempty" json:"url_template,omitempty"`
	Goarm                 string       `yaml:"goarm,omitempty" json:"goarm,omitempty" jsonschema:"oneof_type=string;integer"`
	Goamd64               string       `yaml:"goamd64,omitempty" json:"goamd64,omitempty"`
	SkipUpload            string       `yaml:"skip_upload,omitempty" json:"skip_upload,omitempty" jsonschema:"oneof_type=string;boolean"`

	// Deprecated: use Repository instead.
	Index RepoRef `yaml:"index,omitempty" json:"index,omitempty" jsonschema:"deprecated=true,description=use repository instead"`
}

// Ko contains the ko section
type Ko struct {
	ID                  string            `yaml:"id,omitempty" json:"id,omitempty"`
	Build               string            `yaml:"build,omitempty" json:"build,omitempty"`
	Main                string            `yaml:"main,omitempty" json:"main,omitempty"`
	WorkingDir          string            `yaml:"working_dir,omitempty" json:"working_dir,omitempty"`
	BaseImage           string            `yaml:"base_image,omitempty" json:"base_image,omitempty"`
	Labels              map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Repository          string            `yaml:"repository,omitempty" json:"repository,omitempty"`
	Platforms           []string          `yaml:"platforms,omitempty" json:"platforms,omitempty"`
	Tags                []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
	CreationTime        string            `yaml:"creation_time,omitempty" json:"creation_time,omitempty"`
	KoDataCreationTime  string            `yaml:"ko_data_creation_time,omitempty" json:"ko_data_creation_time,omitempty"`
	SBOM                string            `yaml:"sbom,omitempty" json:"sbom,omitempty"`
	Ldflags             []string          `yaml:"ldflags,omitempty" json:"ldflags,omitempty"`
	Flags               []string          `yaml:"flags,omitempty" json:"flags,omitempty"`
	Env                 []string          `yaml:"env,omitempty" json:"env,omitempty"`
	Bare                bool              `yaml:"bare,omitempty" json:"bare,omitempty"`
	PreserveImportPaths bool              `yaml:"preserve_import_paths,omitempty" json:"preserve_import_paths,omitempty"`
	BaseImportPaths     bool              `yaml:"base_import_paths,omitempty" json:"base_import_paths,omitempty"`
}

// Scoop contains the scoop.sh section.
type Scoop struct {
	Name                  string       `yaml:"name,omitempty" json:"name,omitempty"`
	IDs                   []string     `yaml:"ids,omitempty" json:"ids,omitempty"`
	Repository            RepoRef      `yaml:"repository,omitempty" json:"repository,omitempty"`
	Directory             string       `yaml:"directory,omitempty" json:"directory,omitempty"`
	CommitAuthor          CommitAuthor `yaml:"commit_author,omitempty" json:"commit_author,omitempty"`
	CommitMessageTemplate string       `yaml:"commit_msg_template,omitempty" json:"commit_msg_template,omitempty"`
	Homepage              string       `yaml:"homepage,omitempty" json:"homepage,omitempty"`
	Description           string       `yaml:"description,omitempty" json:"description,omitempty"`
	License               string       `yaml:"license,omitempty" json:"license,omitempty"`
	URLTemplate           string       `yaml:"url_template,omitempty" json:"url_template,omitempty"`
	Persist               []string     `yaml:"persist,omitempty" json:"persist,omitempty"`
	SkipUpload            string       `yaml:"skip_upload,omitempty" json:"skip_upload,omitempty" jsonschema:"oneof_type=string;boolean"`
	PreInstall            []string     `yaml:"pre_install,omitempty" json:"pre_install,omitempty"`
	PostInstall           []string     `yaml:"post_install,omitempty" json:"post_install,omitempty"`
	Depends               []string     `yaml:"depends,omitempty" json:"depends,omitempty"`
	Shortcuts             [][]string   `yaml:"shortcuts,omitempty" json:"shortcuts,omitempty"`
	Goamd64               string       `yaml:"goamd64,omitempty" json:"goamd64,omitempty"`

	// Deprecated: use Repository instead.
	Bucket RepoRef `yaml:"bucket,omitempty" json:"bucket,omitempty" jsonschema:"deprecated=true,description=use repository instead"`

	// Deprecated: use Directory instead.
	Folder string `yaml:"folder,omitempty" json:"folder,omitempty" jsonschema:"deprecated=true"`
}

// CommitAuthor is the author of a Git commit.
type CommitAuthor struct {
	Name  string `yaml:"name,omitempty" json:"name,omitempty"`
	Email string `yaml:"email,omitempty" json:"email,omitempty"`
}

// BuildHooks define actions to run before and/or after something.
type BuildHooks struct { // renamed on pro
	Pre  string `yaml:"pre,omitempty" json:"pre,omitempty"`
	Post string `yaml:"post,omitempty" json:"post,omitempty"`
}

// IgnoredBuild represents a build ignored by the user.
type IgnoredBuild struct {
	Goos    string `yaml:"goos,omitempty" json:"goos,omitempty"`
	Goarch  string `yaml:"goarch,omitempty" json:"goarch,omitempty"`
	Goarm   string `yaml:"goarm,omitempty" json:"goarm,omitempty" jsonschema:"oneof_type=string;integer"`
	Gomips  string `yaml:"gomips,omitempty" json:"gomips,omitempty"`
	Goamd64 string `yaml:"goamd64,omitempty" json:"goamd64,omitempty"`
}

// StringArray is a wrapper for an array of strings.
type StringArray []string

// UnmarshalYAML is a custom unmarshaler that wraps strings in arrays.
func (a *StringArray) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var strings []string
	if err := unmarshal(&strings); err != nil {
		var str string
		if err := unmarshal(&str); err != nil {
			return err
		}
		*a = []string{str}
	} else {
		*a = strings
	}
	return nil
}

func (a StringArray) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{{
			Type: "string",
		}, {
			Type: "array",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		}},
	}
}

// FlagArray is a wrapper for an array of strings.
type FlagArray []string

// UnmarshalYAML is a custom unmarshaler that wraps strings in arrays.
func (a *FlagArray) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var flags []string
	if err := unmarshal(&flags); err != nil {
		var flagstr string
		if err := unmarshal(&flagstr); err != nil {
			return err
		}
		*a = strings.Fields(flagstr)
	} else {
		*a = flags
	}
	return nil
}

func (a FlagArray) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{{
			Type: "string",
		}, {
			Type: "array",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		}},
	}
}

// Build contains the build configuration section.
type Build struct {
	ID              string          `yaml:"id,omitempty" json:"id,omitempty"`
	Goos            []string        `yaml:"goos,omitempty" json:"goos,omitempty"`
	Goarch          []string        `yaml:"goarch,omitempty" json:"goarch,omitempty"`
	Goarm           []string        `yaml:"goarm,omitempty" json:"goarm,omitempty"`
	Gomips          []string        `yaml:"gomips,omitempty" json:"gomips,omitempty"`
	Goamd64         []string        `yaml:"goamd64,omitempty" json:"goamd64,omitempty"`
	Targets         []string        `yaml:"targets,omitempty" json:"targets,omitempty"`
	Ignore          []IgnoredBuild  `yaml:"ignore,omitempty" json:"ignore,omitempty"`
	Dir             string          `yaml:"dir,omitempty" json:"dir,omitempty"`
	Main            string          `yaml:"main,omitempty" json:"main,omitempty"`
	Binary          string          `yaml:"binary,omitempty" json:"binary,omitempty"`
	Hooks           BuildHookConfig `yaml:"hooks,omitempty" json:"hooks,omitempty"`
	Builder         string          `yaml:"builder,omitempty" json:"builder,omitempty"`
	ModTimestamp    string          `yaml:"mod_timestamp,omitempty" json:"mod_timestamp,omitempty"`
	Skip            bool            `yaml:"skip,omitempty" json:"skip,omitempty"`
	GoBinary        string          `yaml:"gobinary,omitempty" json:"gobinary,omitempty"`
	Command         string          `yaml:"command,omitempty" json:"command,omitempty"`
	NoUniqueDistDir bool            `yaml:"no_unique_dist_dir,omitempty" json:"no_unique_dist_dir,omitempty"`
	NoMainCheck     bool            `yaml:"no_main_check,omitempty" json:"no_main_check,omitempty"`
	UnproxiedMain   string          `yaml:"-" json:"-"` // used by gomod.proxy
	UnproxiedDir    string          `yaml:"-" json:"-"` // used by gomod.proxy

	BuildDetails          `yaml:",inline" json:",inline"` // nolint: tagliatelle
	BuildDetailsOverrides []BuildDetailsOverride          `yaml:"overrides,omitempty" json:"overrides,omitempty"`
}

type BuildDetailsOverride struct {
	Goos         string                          `yaml:"goos,omitempty" json:"goos,omitempty"`
	Goarch       string                          `yaml:"goarch,omitempty" json:"goarch,omitempty"`
	Goarm        string                          `yaml:"goarm,omitempty" json:"goarm,omitempty" jsonschema:"oneof_type=string;integer"`
	Gomips       string                          `yaml:"gomips,omitempty" json:"gomips,omitempty"`
	Goamd64      string                          `yaml:"goamd64,omitempty" json:"goamd64,omitempty"`
	BuildDetails `yaml:",inline" json:",inline"` // nolint: tagliatelle
}

type BuildDetails struct {
	Buildmode string      `yaml:"buildmode,omitempty" json:"buildmode,omitempty" jsonschema:"enum=c-archive,enum=c-shared,enum=pie,enum=,default="`
	Ldflags   StringArray `yaml:"ldflags,omitempty" json:"ldflags,omitempty"`
	Tags      FlagArray   `yaml:"tags,omitempty" json:"tags,omitempty"`
	Flags     FlagArray   `yaml:"flags,omitempty" json:"flags,omitempty"`
	Asmflags  StringArray `yaml:"asmflags,omitempty" json:"asmflags,omitempty"`
	Gcflags   StringArray `yaml:"gcflags,omitempty" json:"gcflags,omitempty"`
	Env       []string    `yaml:"env,omitempty" json:"env,omitempty"`
}

type BuildHookConfig struct {
	Pre  Hooks `yaml:"pre,omitempty" json:"pre,omitempty"`
	Post Hooks `yaml:"post,omitempty" json:"post,omitempty"`
}

type Hooks []Hook

// UnmarshalYAML is a custom unmarshaler that allows simplified declaration of single command.
func (bhc *Hooks) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var singleCmd string
	err := unmarshal(&singleCmd)
	if err == nil {
		*bhc = []Hook{{Cmd: singleCmd}}
		return nil
	}

	type t Hooks
	var hooks t
	if err := unmarshal(&hooks); err != nil {
		return err
	}
	*bhc = (Hooks)(hooks)
	return nil
}

func (bhc Hooks) JSONSchema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	var t Hook
	schema := reflector.Reflect(&t)
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{{
			Type: "string",
		}, {
			Type:  "array",
			Items: schema,
		}},
	}
}

type Hook struct {
	Dir    string   `yaml:"dir,omitempty" json:"dir,omitempty"`
	Cmd    string   `yaml:"cmd,omitempty" json:"cmd,omitempty"`
	Env    []string `yaml:"env,omitempty" json:"env,omitempty"`
	Output bool     `yaml:"output,omitempty" json:"output,omitempty"`
}

// UnmarshalYAML is a custom unmarshaler that allows simplified declarations of commands as strings.
func (bh *Hook) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var cmd string
	if err := unmarshal(&cmd); err != nil {
		type t Hook
		var hook t
		if err := unmarshal(&hook); err != nil {
			return err
		}
		*bh = (Hook)(hook)
		return nil
	}

	bh.Cmd = cmd
	return nil
}

func (bh Hook) JSONSchema() *jsonschema.Schema {
	type hookAlias Hook
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	schema := reflector.Reflect(&hookAlias{})
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{
				Type: "string",
			},
			schema,
		},
	}
}

// FormatOverride is used to specify a custom format for a specific GOOS.
type FormatOverride struct {
	Goos   string `yaml:"goos,omitempty" json:"goos,omitempty"`
	Format string `yaml:"format,omitempty" json:"format,omitempty" jsonschema:"enum=tar,enum=tgz,enum=tar.gz,enum=zip,enum=gz,enum=tar.xz,enum=txz,enum=binary,enum=none,default=tar.gz"`
}

// File is a file inside an archive.
type File struct {
	Source      string   `yaml:"src,omitempty" json:"src,omitempty"`
	Destination string   `yaml:"dst,omitempty" json:"dst,omitempty"`
	StripParent bool     `yaml:"strip_parent,omitempty" json:"strip_parent,omitempty"`
	Info        FileInfo `yaml:"info,omitempty" json:"info,omitempty"`
	Default     bool     `yaml:"-" json:"-"`
}

// FileInfo is the file info of a file.
type FileInfo struct {
	Owner       string      `yaml:"owner,omitempty" json:"owner,omitempty"`
	Group       string      `yaml:"group,omitempty" json:"group,omitempty"`
	Mode        os.FileMode `yaml:"mode,omitempty" json:"mode,omitempty"`
	MTime       string      `yaml:"mtime,omitempty" json:"mtime,omitempty"`
	ParsedMTime time.Time   `yaml:"-" json:"-"`
}

// UnmarshalYAML is a custom unmarshaler that wraps strings in arrays.
func (f *File) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type t File
	var str string
	if err := unmarshal(&str); err == nil {
		*f = File{Source: str}
		return nil
	}

	var file t
	if err := unmarshal(&file); err != nil {
		return err
	}
	*f = File(file)
	return nil
}

func (f File) JSONSchema() *jsonschema.Schema {
	type fileAlias File
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	schema := reflector.Reflect(&fileAlias{})
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{
				Type: "string",
			},
			schema,
		},
	}
}

// UniversalBinary setups macos universal binaries.
type UniversalBinary struct {
	ID           string          `yaml:"id,omitempty" json:"id,omitempty"`
	IDs          []string        `yaml:"ids,omitempty" json:"ids,omitempty"`
	NameTemplate string          `yaml:"name_template,omitempty" json:"name_template,omitempty"`
	Replace      bool            `yaml:"replace,omitempty" json:"replace,omitempty"`
	Hooks        BuildHookConfig `yaml:"hooks,omitempty" json:"hooks,omitempty"`
	ModTimestamp string          `yaml:"mod_timestamp,omitempty" json:"mod_timestamp,omitempty"`
}

// UPX allows to compress binaries with `upx`.
type UPX struct {
	Enabled  string   `yaml:"enabled,omitempty" json:"enabled,omitempty" jsonschema:"oneof_type=string;boolean"`
	IDs      []string `yaml:"ids,omitempty" json:"ids,omitempty"`
	Goos     []string `yaml:"goos,omitempty" json:"goos,omitempty"`
	Goarch   []string `yaml:"goarch,omitempty" json:"goarch,omitempty"`
	Goarm    []string `yaml:"goarm,omitempty" json:"goarm,omitempty"`
	Goamd64  []string `yaml:"goamd64,omitempty" json:"goamd64,omitempty"`
	Binary   string   `yaml:"binary,omitempty" json:"binary,omitempty"`
	Compress string   `yaml:"compress,omitempty" json:"compress,omitempty" jsonschema:"enum=1,enum=2,enum=3,enum=4,enum=5,enum=6,enum=7,enum=8,enum=9,enum=best,enum=,default="`
	LZMA     bool     `yaml:"lzma,omitempty" json:"lzma,omitempty"`
	Brute    bool     `yaml:"brute,omitempty" json:"brute,omitempty"`
}

// Archive config used for the archive.
type Archive struct {
	ID                        string           `yaml:"id,omitempty" json:"id,omitempty"`
	Builds                    []string         `yaml:"builds,omitempty" json:"builds,omitempty"`
	BuildsInfo                FileInfo         `yaml:"builds_info,omitempty" json:"builds_info,omitempty"`
	NameTemplate              string           `yaml:"name_template,omitempty" json:"name_template,omitempty"`
	Format                    string           `yaml:"format,omitempty" json:"format,omitempty" jsonschema:"enum=tar,enum=tgz,enum=tar.gz,enum=zip,enum=gz,enum=tar.xz,enum=txz,enum=binary,default=tar.gz"`
	FormatOverrides           []FormatOverride `yaml:"format_overrides,omitempty" json:"format_overrides,omitempty"`
	WrapInDirectory           string           `yaml:"wrap_in_directory,omitempty" json:"wrap_in_directory,omitempty" jsonschema:"oneof_type=string;boolean"`
	StripBinaryDirectory      bool             `yaml:"strip_binary_directory,omitempty" json:"strip_binary_directory,omitempty"`
	Files                     []File           `yaml:"files,omitempty" json:"files,omitempty"`
	Meta                      bool             `yaml:"meta,omitempty" json:"meta,omitempty"`
	AllowDifferentBinaryCount bool             `yaml:"allow_different_binary_count,omitempty" json:"allow_different_binary_count,omitempty"`

	// Deprecated: don't need to set this anymore.
	RLCP string `yaml:"rlcp,omitempty" json:"rlcp,omitempty"  jsonschema:"oneof_type=string;boolean,deprecated=true,description=you can now remove this"`

	// Deprecated: use StripBinaryDirectory instead.
	StripParentBinaryFolder bool `yaml:"strip_parent_binary_folder,omitempty" json:"strip_parent_binary_folder,omitempty" jsonschema:"deprecated=true"`
}

type ReleaseNotesMode string

const (
	ReleaseNotesModeKeepExisting ReleaseNotesMode = "keep-existing"
	ReleaseNotesModeAppend       ReleaseNotesMode = "append"
	ReleaseNotesModeReplace      ReleaseNotesMode = "replace"
	ReleaseNotesModePrepend      ReleaseNotesMode = "prepend"
)

// Release config used for the GitHub/GitLab release.
type Release struct {
	GitHub                 Repo        `yaml:"github,omitempty" json:"github,omitempty"`
	GitLab                 Repo        `yaml:"gitlab,omitempty" json:"gitlab,omitempty"`
	Gitea                  Repo        `yaml:"gitea,omitempty" json:"gitea,omitempty"`
	Draft                  bool        `yaml:"draft,omitempty" json:"draft,omitempty"`
	ReplaceExistingDraft   bool        `yaml:"replace_existing_draft,omitempty" json:"replace_existing_draft,omitempty"`
	TargetCommitish        string      `yaml:"target_commitish,omitempty" json:"target_commitish,omitempty"`
	Disable                string      `yaml:"disable,omitempty" json:"disable,omitempty" jsonschema:"oneof_type=string;boolean"`
	SkipUpload             string      `yaml:"skip_upload,omitempty" json:"skip_upload,omitempty" jsonschema:"oneof_type=string;boolean"`
	Prerelease             string      `yaml:"prerelease,omitempty" json:"prerelease,omitempty"`
	MakeLatest             string      `yaml:"make_latest,omitempty" json:"make_latest,omitempty" jsonschema:"oneof_type=string;boolean"`
	NameTemplate           string      `yaml:"name_template,omitempty" json:"name_template,omitempty"`
	IDs                    []string    `yaml:"ids,omitempty" json:"ids,omitempty"`
	ExtraFiles             []ExtraFile `yaml:"extra_files,omitempty" json:"extra_files,omitempty"`
	DiscussionCategoryName string      `yaml:"discussion_category_name,omitempty" json:"discussion_category_name,omitempty"`
	Header                 string      `yaml:"header,omitempty" json:"header,omitempty"`
	Footer                 string      `yaml:"footer,omitempty" json:"footer,omitempty"`

	ReleaseNotesMode         ReleaseNotesMode `yaml:"mode,omitempty" json:"mode,omitempty" jsonschema:"enum=keep-existing,enum=append,enum=prepend,enum=replace,default=keep-existing"`
	ReplaceExistingArtifacts bool             `yaml:"replace_existing_artifacts,omitempty" json:"replace_existing_artifacts,omitempty"`
	IncludeMeta              bool             `yaml:"include_meta,omitempty" json:"include_meta,omitempty"`
}

// Milestone config used for VCS milestone.
type Milestone struct {
	Repo         Repo   `yaml:"repo,omitempty" json:"repo,omitempty"`
	Close        bool   `yaml:"close,omitempty" json:"close,omitempty"`
	FailOnError  bool   `yaml:"fail_on_error,omitempty" json:"fail_on_error,omitempty"`
	NameTemplate string `yaml:"name_template,omitempty" json:"name_template,omitempty"`
}

// ExtraFile on a release.
type ExtraFile struct {
	Glob         string `yaml:"glob,omitempty" json:"glob,omitempty"`
	NameTemplate string `yaml:"name_template,omitempty" json:"name_template,omitempty"`
}

// NFPM config.
type NFPM struct {
	NFPMOverridables `yaml:",inline" json:",inline"` // nolint: tagliatelle
	Overrides        map[string]NFPMOverridables     `yaml:"overrides,omitempty" json:"overrides,omitempty"`

	ID          string   `yaml:"id,omitempty" json:"id,omitempty"`
	Builds      []string `yaml:"builds,omitempty" json:"builds,omitempty"`
	Formats     []string `yaml:"formats,omitempty" json:"formats,omitempty" jsonschema:"enum=apk,enum=deb,enum=rpm,enum=termux.deb,enum=archlinux"`
	Section     string   `yaml:"section,omitempty" json:"section,omitempty"`
	Priority    string   `yaml:"priority,omitempty" json:"priority,omitempty"`
	Vendor      string   `yaml:"vendor,omitempty" json:"vendor,omitempty"`
	Homepage    string   `yaml:"homepage,omitempty" json:"homepage,omitempty"`
	Maintainer  string   `yaml:"maintainer,omitempty" json:"maintainer,omitempty"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	License     string   `yaml:"license,omitempty" json:"license,omitempty"`
	Bindir      string   `yaml:"bindir,omitempty" json:"bindir,omitempty"`
	Libdirs     Libdirs  `yaml:"libdirs,omitempty" json:"libdirs,omitempty"`
	Changelog   string   `yaml:"changelog,omitempty" json:"changelog,omitempty"`
	Meta        bool     `yaml:"meta,omitempty" json:"meta,omitempty"` // make package without binaries - only deps
}

type Libdirs struct {
	Header   string `yaml:"header,omitempty" json:"header,omitempty"`
	CArchive string `yaml:"carchive,omitempty" json:"carchive,omitempty"`
	CShared  string `yaml:"cshared,omitempty" json:"cshared,omitempty"`
}

// NFPMScripts is used to specify maintainer scripts.
type NFPMScripts struct {
	PreInstall  string `yaml:"preinstall,omitempty" json:"preinstall,omitempty"`
	PostInstall string `yaml:"postinstall,omitempty" json:"postinstall,omitempty"`
	PreRemove   string `yaml:"preremove,omitempty" json:"preremove,omitempty"`
	PostRemove  string `yaml:"postremove,omitempty" json:"postremove,omitempty"`
}

type NFPMRPMSignature struct {
	// PGP secret key, can be ASCII-armored
	KeyFile       string `yaml:"key_file,omitempty" json:"key_file,omitempty"`
	KeyPassphrase string `yaml:"-" json:"-"` // populated from environment variable
}

// NFPMRPMScripts represents scripts only available on RPM packages.
type NFPMRPMScripts struct {
	PreTrans  string `yaml:"pretrans,omitempty" json:"pretrans,omitempty"`
	PostTrans string `yaml:"posttrans,omitempty" json:"posttrans,omitempty"`
}

// NFPMRPM is custom configs that are only available on RPM packages.
type NFPMRPM struct {
	Summary     string           `yaml:"summary,omitempty" json:"summary,omitempty"`
	Group       string           `yaml:"group,omitempty" json:"group,omitempty"`
	Compression string           `yaml:"compression,omitempty" json:"compression,omitempty"`
	Signature   NFPMRPMSignature `yaml:"signature,omitempty" json:"signature,omitempty"`
	Scripts     NFPMRPMScripts   `yaml:"scripts,omitempty" json:"scripts,omitempty"`
	Prefixes    []string         `yaml:"prefixes,omitempty" json:"prefixes,omitempty"`
	Packager    string           `yaml:"packager,omitempty" json:"packager,omitempty"`
}

// NFPMDebScripts is scripts only available on deb packages.
type NFPMDebScripts struct {
	Rules     string `yaml:"rules,omitempty" json:"rules,omitempty"`
	Templates string `yaml:"templates,omitempty" json:"templates,omitempty"`
}

// NFPMDebTriggers contains triggers only available for deb packages.
// https://wiki.debian.org/DpkgTriggers
// https://man7.org/linux/man-pages/man5/deb-triggers.5.html
type NFPMDebTriggers struct {
	Interest        []string `yaml:"interest,omitempty" json:"interest,omitempty"`
	InterestAwait   []string `yaml:"interest_await,omitempty" json:"interest_await,omitempty"`
	InterestNoAwait []string `yaml:"interest_noawait,omitempty" json:"interest_noawait,omitempty"`
	Activate        []string `yaml:"activate,omitempty" json:"activate,omitempty"`
	ActivateAwait   []string `yaml:"activate_await,omitempty" json:"activate_await,omitempty"`
	ActivateNoAwait []string `yaml:"activate_noawait,omitempty" json:"activate_noawait,omitempty"`
}

// NFPMDebSignature contains config for signing deb packages created by nfpm.
type NFPMDebSignature struct {
	// PGP secret key, can be ASCII-armored
	KeyFile       string `yaml:"key_file,omitempty" json:"key_file,omitempty"`
	KeyPassphrase string `yaml:"-" json:"-"` // populated from environment variable
	// origin, maint or archive (defaults to origin)
	Type string `yaml:"type,omitempty" json:"type,omitempty"`
}

// NFPMDeb is custom configs that are only available on deb packages.
type NFPMDeb struct {
	Scripts     NFPMDebScripts    `yaml:"scripts,omitempty" json:"scripts,omitempty"`
	Triggers    NFPMDebTriggers   `yaml:"triggers,omitempty" json:"triggers,omitempty"`
	Breaks      []string          `yaml:"breaks,omitempty" json:"breaks,omitempty"`
	Signature   NFPMDebSignature  `yaml:"signature,omitempty" json:"signature,omitempty"`
	Lintian     []string          `yaml:"lintian_overrides,omitempty" json:"lintian_overrides,omitempty"`
	Compression string            `yaml:"compression,omitempty" json:"compression,omitempty" jsonschema:"enum=gzip,enum=xz,enum=none,default=gzip"`
	Fields      map[string]string `yaml:"fields,omitempty" json:"fields,omitempty"`
	Predepends  []string          `yaml:"predepends,omitempty" json:"predepends,omitempty"`
}

type NFPMAPKScripts struct {
	PreUpgrade  string `yaml:"preupgrade,omitempty" json:"preupgrade,omitempty"`
	PostUpgrade string `yaml:"postupgrade,omitempty" json:"postupgrade,omitempty"`
}

// NFPMAPKSignature contains config for signing apk packages created by nfpm.
type NFPMAPKSignature struct {
	// RSA private key in PEM format
	KeyFile       string `yaml:"key_file,omitempty" json:"key_file,omitempty"`
	KeyPassphrase string `yaml:"-" json:"-"` // populated from environment variable
	// defaults to <maintainer email>.rsa.pub
	KeyName string `yaml:"key_name,omitempty" json:"key_name,omitempty"`
}

// NFPMAPK is custom config only available on apk packages.
type NFPMAPK struct {
	Scripts   NFPMAPKScripts   `yaml:"scripts,omitempty" json:"scripts,omitempty"`
	Signature NFPMAPKSignature `yaml:"signature,omitempty" json:"signature,omitempty"`
}

type NFPMArchLinuxScripts struct {
	PreUpgrade  string `yaml:"preupgrade,omitempty" json:"preupgrade,omitempty"`
	PostUpgrade string `yaml:"postupgrade,omitempty" json:"postupgrade,omitempty"`
}

type NFPMArchLinux struct {
	Pkgbase  string               `yaml:"pkgbase,omitempty" json:"pkgbase,omitempty"`
	Packager string               `yaml:"packager,omitempty" json:"packager,omitempty"`
	Scripts  NFPMArchLinuxScripts `yaml:"scripts,omitempty" json:"scripts,omitempty"`
}

// NFPMOverridables is used to specify per package format settings.
type NFPMOverridables struct {
	FileNameTemplate string         `yaml:"file_name_template,omitempty" json:"file_name_template,omitempty"`
	PackageName      string         `yaml:"package_name,omitempty" json:"package_name,omitempty"`
	Epoch            string         `yaml:"epoch,omitempty" json:"epoch,omitempty"`
	Release          string         `yaml:"release,omitempty" json:"release,omitempty"`
	Prerelease       string         `yaml:"prerelease,omitempty" json:"prerelease,omitempty"`
	VersionMetadata  string         `yaml:"version_metadata,omitempty" json:"version_metadata,omitempty"`
	Dependencies     []string       `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Recommends       []string       `yaml:"recommends,omitempty" json:"recommends,omitempty"`
	Suggests         []string       `yaml:"suggests,omitempty" json:"suggests,omitempty"`
	Conflicts        []string       `yaml:"conflicts,omitempty" json:"conflicts,omitempty"`
	Umask            fs.FileMode    `yaml:"umask,omitempty" json:"umask,omitempty"`
	Replaces         []string       `yaml:"replaces,omitempty" json:"replaces,omitempty"`
	Provides         []string       `yaml:"provides,omitempty" json:"provides,omitempty"`
	Contents         files.Contents `yaml:"contents,omitempty" json:"contents,omitempty"`
	Scripts          NFPMScripts    `yaml:"scripts,omitempty" json:"scripts,omitempty"`
	RPM              NFPMRPM        `yaml:"rpm,omitempty" json:"rpm,omitempty"`
	Deb              NFPMDeb        `yaml:"deb,omitempty" json:"deb,omitempty"`
	APK              NFPMAPK        `yaml:"apk,omitempty" json:"apk,omitempty"`
	ArchLinux        NFPMArchLinux  `yaml:"archlinux,omitempty" json:"archlinux,omitempty"`
}

// SBOM config.
type SBOM struct {
	ID        string   `yaml:"id,omitempty" json:"id,omitempty"`
	Cmd       string   `yaml:"cmd,omitempty" json:"cmd,omitempty"`
	Env       []string `yaml:"env,omitempty" json:"env,omitempty"`
	Args      []string `yaml:"args,omitempty" json:"args,omitempty"`
	Documents []string `yaml:"documents,omitempty" json:"documents,omitempty"`
	Artifacts string   `yaml:"artifacts,omitempty" json:"artifacts,omitempty" jsonschema:"enum=source,enum=package,enum=archive,enum=binary,enum=any,enum=any,default=archive"`
	IDs       []string `yaml:"ids,omitempty" json:"ids,omitempty"`
}

// Sign config.
type Sign struct {
	ID          string   `yaml:"id,omitempty" json:"id,omitempty"`
	Cmd         string   `yaml:"cmd,omitempty" json:"cmd,omitempty"`
	Args        []string `yaml:"args,omitempty" json:"args,omitempty"`
	Signature   string   `yaml:"signature,omitempty" json:"signature,omitempty"`
	Artifacts   string   `yaml:"artifacts,omitempty" json:"artifacts,omitempty" jsonschema:"enum=all,enum=manifests,enum=images,enum=checksum,enum=source,enum=package,enum=archive,enum=binary,enum=sbom"`
	IDs         []string `yaml:"ids,omitempty" json:"ids,omitempty"`
	Stdin       *string  `yaml:"stdin,omitempty" json:"stdin,omitempty"`
	StdinFile   string   `yaml:"stdin_file,omitempty" json:"stdin_file,omitempty"`
	Env         []string `yaml:"env,omitempty" json:"env,omitempty"`
	Certificate string   `yaml:"certificate,omitempty" json:"certificate,omitempty"`
	Output      bool     `yaml:"output,omitempty" json:"output,omitempty"`
}

// SnapcraftAppMetadata for the binaries that will be in the snap package.
type SnapcraftAppMetadata struct {
	Command string `yaml:"command" json:"command"`
	Args    string `yaml:"args,omitempty" json:"args,omitempty"`

	Adapter          string                 `yaml:"adapter,omitempty" json:"adapter,omitempty"`
	After            []string               `yaml:"after,omitempty" json:"after,omitempty"`
	Aliases          []string               `yaml:"aliases,omitempty" json:"aliases,omitempty"`
	Autostart        string                 `yaml:"autostart,omitempty" json:"autostart,omitempty"`
	Before           []string               `yaml:"before,omitempty" json:"before,omitempty"`
	BusName          string                 `yaml:"bus_name,omitempty" json:"bus_name,omitempty"`
	CommandChain     []string               `yaml:"command_chain,omitempty" json:"command_chain,omitempty"`
	CommonID         string                 `yaml:"common_id,omitempty" json:"common_id,omitempty"`
	Completer        string                 `yaml:"completer,omitempty" json:"completer,omitempty"`
	Daemon           string                 `yaml:"daemon,omitempty" json:"daemon,omitempty"`
	Desktop          string                 `yaml:"desktop,omitempty" json:"desktop,omitempty"`
	Environment      map[string]interface{} `yaml:"environment,omitempty" json:"environment,omitempty"`
	Extensions       []string               `yaml:"extensions,omitempty" json:"extensions,omitempty"`
	InstallMode      string                 `yaml:"install_mode,omitempty" json:"install_mode,omitempty"`
	Passthrough      map[string]interface{} `yaml:"passthrough,omitempty" json:"passthrough,omitempty"`
	Plugs            []string               `yaml:"plugs,omitempty" json:"plugs,omitempty"`
	PostStopCommand  string                 `yaml:"post_stop_command,omitempty" json:"post_stop_command,omitempty"`
	RefreshMode      string                 `yaml:"refresh_mode,omitempty" json:"refresh_mode,omitempty"`
	ReloadCommand    string                 `yaml:"reload_command,omitempty" json:"reload_command,omitempty"`
	RestartCondition string                 `yaml:"restart_condition,omitempty" json:"restart_condition,omitempty"`
	RestartDelay     string                 `yaml:"restart_delay,omitempty" json:"restart_delay,omitempty"`
	Slots            []string               `yaml:"slots,omitempty" json:"slots,omitempty"`
	Sockets          map[string]interface{} `yaml:"sockets,omitempty" json:"sockets,omitempty"`
	StartTimeout     string                 `yaml:"start_timeout,omitempty" json:"start_timeout,omitempty"`
	StopCommand      string                 `yaml:"stop_command,omitempty" json:"stop_command,omitempty"`
	StopMode         string                 `yaml:"stop_mode,omitempty" json:"stop_mode,omitempty"`
	StopTimeout      string                 `yaml:"stop_timeout,omitempty" json:"stop_timeout,omitempty"`
	Timer            string                 `yaml:"timer,omitempty" json:"timer,omitempty"`
	WatchdogTimeout  string                 `yaml:"watchdog_timeout,omitempty" json:"watchdog_timeout,omitempty"`
}

type SnapcraftLayoutMetadata struct {
	Symlink  string `yaml:"symlink,omitempty" json:"symlink,omitempty"`
	Bind     string `yaml:"bind,omitempty" json:"bind,omitempty"`
	BindFile string `yaml:"bind_file,omitempty" json:"bind_file,omitempty"`
	Type     string `yaml:"type,omitempty" json:"type,omitempty"`
}

// Snapcraft config.
type Snapcraft struct {
	NameTemplate     string                             `yaml:"name_template,omitempty" json:"name_template,omitempty"`
	Publish          bool                               `yaml:"publish,omitempty" json:"publish,omitempty"`
	ID               string                             `yaml:"id,omitempty" json:"id,omitempty"`
	Builds           []string                           `yaml:"builds,omitempty" json:"builds,omitempty"`
	Name             string                             `yaml:"name,omitempty" json:"name,omitempty"`
	Title            string                             `yaml:"title,omitempty" json:"title,omitempty"`
	Summary          string                             `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description      string                             `yaml:"description,omitempty" json:"description,omitempty"`
	Icon             string                             `yaml:"icon,omitempty" json:"icon,omitempty"`
	Base             string                             `yaml:"base,omitempty" json:"base,omitempty"`
	License          string                             `yaml:"license,omitempty" json:"license,omitempty"`
	Grade            string                             `yaml:"grade,omitempty" json:"grade,omitempty"`
	ChannelTemplates []string                           `yaml:"channel_templates,omitempty" json:"channel_templates,omitempty"`
	Confinement      string                             `yaml:"confinement,omitempty" json:"confinement,omitempty"`
	Assumes          []string                           `yaml:"assumes,omitempty" json:"assumes,omitempty"`
	Layout           map[string]SnapcraftLayoutMetadata `yaml:"layout,omitempty" json:"layout,omitempty"`
	Apps             map[string]SnapcraftAppMetadata    `yaml:"apps,omitempty" json:"apps,omitempty"`
	Hooks            map[string]interface{}             `yaml:"hooks,omitempty" json:"hooks,omitempty"`
	Plugs            map[string]interface{}             `yaml:"plugs,omitempty" json:"plugs,omitempty"`
	Disable          string                             `yaml:"disable,omitempty" json:"disable,omitempty" jsonschema:"oneof_type=string;boolean"`

	Files []SnapcraftExtraFiles `yaml:"extra_files,omitempty" json:"extra_files,omitempty"`
}

// SnapcraftExtraFiles config.
type SnapcraftExtraFiles struct {
	Source      string `yaml:"source" json:"source"`
	Destination string `yaml:"destination,omitempty" json:"destination,omitempty"`
	Mode        uint32 `yaml:"mode,omitempty" json:"mode,omitempty"`
}

// Snapshot config.
type Snapshot struct {
	NameTemplate string `yaml:"name_template,omitempty" json:"name_template,omitempty"`
}

// Checksum config.
type Checksum struct {
	NameTemplate string      `yaml:"name_template,omitempty" json:"name_template,omitempty"`
	Algorithm    string      `yaml:"algorithm,omitempty" json:"algorithm,omitempty"`
	Split        bool        `yaml:"split,omitempty" json:"split,omitempty"`
	IDs          []string    `yaml:"ids,omitempty" json:"ids,omitempty"`
	Disable      bool        `yaml:"disable,omitempty" json:"disable,omitempty"`
	ExtraFiles   []ExtraFile `yaml:"extra_files,omitempty" json:"extra_files,omitempty"`
}

// Docker image config.
type Docker struct {
	ID                 string   `yaml:"id,omitempty" json:"id,omitempty"`
	IDs                []string `yaml:"ids,omitempty" json:"ids,omitempty"`
	Goos               string   `yaml:"goos,omitempty" json:"goos,omitempty"`
	Goarch             string   `yaml:"goarch,omitempty" json:"goarch,omitempty"`
	Goarm              string   `yaml:"goarm,omitempty" json:"goarm,omitempty" jsonschema:"oneof_type=string;integer"`
	Goamd64            string   `yaml:"goamd64,omitempty" json:"goamd64,omitempty"`
	Dockerfile         string   `yaml:"dockerfile,omitempty" json:"dockerfile,omitempty"`
	ImageTemplates     []string `yaml:"image_templates,omitempty" json:"image_templates,omitempty"`
	SkipPush           string   `yaml:"skip_push,omitempty" json:"skip_push,omitempty" jsonschema:"oneof_type=string;boolean"`
	Files              []string `yaml:"extra_files,omitempty" json:"extra_files,omitempty"`
	BuildFlagTemplates []string `yaml:"build_flag_templates,omitempty" json:"build_flag_templates,omitempty"`
	PushFlags          []string `yaml:"push_flags,omitempty" json:"push_flags,omitempty"`
	Use                string   `yaml:"use,omitempty" json:"use,omitempty" jsonschema:"enum=docker,enum=buildx,default=docker"`
}

// DockerManifest config.
type DockerManifest struct {
	ID             string   `yaml:"id,omitempty" json:"id,omitempty"`
	NameTemplate   string   `yaml:"name_template,omitempty" json:"name_template,omitempty"`
	SkipPush       string   `yaml:"skip_push,omitempty" json:"skip_push,omitempty" jsonschema:"oneof_type=string;boolean"`
	ImageTemplates []string `yaml:"image_templates,omitempty" json:"image_templates,omitempty"`
	CreateFlags    []string `yaml:"create_flags,omitempty" json:"create_flags,omitempty"`
	PushFlags      []string `yaml:"push_flags,omitempty" json:"push_flags,omitempty"`
	Use            string   `yaml:"use,omitempty" json:"use,omitempty"`
}

// Filters config.
type Filters struct {
	Include []string `yaml:"include,omitempty" json:"include,omitempty"`
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

// Changelog Config.
type Changelog struct {
	Filters Filters          `yaml:"filters,omitempty" json:"filters,omitempty"`
	Sort    string           `yaml:"sort,omitempty" json:"sort,omitempty" jsonschema:"enum=asc,enum=desc,enum=,default="`
	Disable string           `yaml:"disable,omitempty" json:"disable,omitempty" jsonschema:"oneof_type=string;boolean"`
	Use     string           `yaml:"use,omitempty" json:"use,omitempty" jsonschema:"enum=git,enum=github,enum=github-native,enum=gitlab,default=git"`
	Groups  []ChangelogGroup `yaml:"groups,omitempty" json:"groups,omitempty"`
	Abbrev  int              `yaml:"abbrev,omitempty" json:"abbrev,omitempty"`

	// Deprecated: use disable instead.
	Skip string `yaml:"skip,omitempty" json:"skip,omitempty" jsonschema:"oneof_type=string;boolean,deprecated=true,description=use disable instead"`
}

// ChangelogGroup holds the grouping criteria for the changelog.
type ChangelogGroup struct {
	Title  string `yaml:"title,omitempty" json:"title,omitempty"`
	Regexp string `yaml:"regexp,omitempty" json:"regexp,omitempty"`
	Order  int    `yaml:"order,omitempty" json:"order,omitempty"`
}

// EnvFiles holds paths to files that contains environment variables
// values like the github token for example.
type EnvFiles struct {
	GitHubToken string `yaml:"github_token,omitempty" json:"github_token,omitempty"`
	GitLabToken string `yaml:"gitlab_token,omitempty" json:"gitlab_token,omitempty"`
	GiteaToken  string `yaml:"gitea_token,omitempty" json:"gitea_token,omitempty"`
}

// Before config.
type Before struct {
	Hooks []string `yaml:"hooks,omitempty" json:"hooks,omitempty"`
}

// Blob contains config for GO CDK blob.
type Blob struct {
	Bucket             string      `yaml:"bucket,omitempty" json:"bucket,omitempty"`
	Provider           string      `yaml:"provider,omitempty" json:"provider,omitempty"`
	Region             string      `yaml:"region,omitempty" json:"region,omitempty"`
	DisableSSL         bool        `yaml:"disable_ssl,omitempty" json:"disable_ssl,omitempty"`
	Directory          string      `yaml:"directory,omitempty" json:"directory,omitempty"`
	KMSKey             string      `yaml:"kms_key,omitempty" json:"kms_key,omitempty"`
	IDs                []string    `yaml:"ids,omitempty" json:"ids,omitempty"`
	Endpoint           string      `yaml:"endpoint,omitempty" json:"endpoint,omitempty"` // used for minio for example
	ExtraFiles         []ExtraFile `yaml:"extra_files,omitempty" json:"extra_files,omitempty"`
	Disable            string      `yaml:"disable,omitempty" json:"disable,omitempty" jsonschema:"oneof_type=string;boolean"`
	S3ForcePathStyle   *bool       `yaml:"s3_force_path_style,omitempty" json:"s3_force_path_style,omitempty"`
	ACL                string      `yaml:"acl,omitempty" json:"acl,omitempty"`
	CacheControl       []string    `yaml:"cache_control,omitempty" json:"cache_control,omitempty"`
	ContentDisposition string      `yaml:"content_disposition,omitempty" json:"content_disposition,omitempty"`
	IncludeMeta        bool        `yaml:"include_meta,omitempty" json:"include_meta,omitempty"`

	// Deprecated: use disable_ssl instead
	OldDisableSSL bool `yaml:"disableSSL,omitempty" json:"disableSSL,omitempty" jsonschema:"deprecated=true,description=use disable_ssl instead"` // nolint:tagliatelle

	// Deprecated: use kms_key instead
	OldKMSKey string `yaml:"kmskey,omitempty" json:"kmskey,omitempty" jsonschema:"deprecated=true,description=use kms_key instead"`

	// Deprecated: use Directory instead.
	Folder string `yaml:"folder,omitempty" json:"folder,omitempty" jsonschema:"deprecated=true"`
}

// Upload configuration.
type Upload struct {
	Name               string            `yaml:"name,omitempty" json:"name,omitempty"`
	IDs                []string          `yaml:"ids,omitempty" json:"ids,omitempty"`
	Exts               []string          `yaml:"exts,omitempty" json:"exts,omitempty"`
	Target             string            `yaml:"target,omitempty" json:"target,omitempty"`
	Username           string            `yaml:"username,omitempty" json:"username,omitempty"`
	Mode               string            `yaml:"mode,omitempty" json:"mode,omitempty"`
	Method             string            `yaml:"method,omitempty" json:"method,omitempty"`
	ChecksumHeader     string            `yaml:"checksum_header,omitempty" json:"checksum_header,omitempty"`
	ClientX509Cert     string            `yaml:"client_x509_cert,omitempty" json:"client_x509_cert,omitempty"`
	ClientX509Key      string            `yaml:"client_x509_key,omitempty" json:"client_x509_key,omitempty"`
	TrustedCerts       string            `yaml:"trusted_certificates,omitempty" json:"trusted_certificates,omitempty"`
	Checksum           bool              `yaml:"checksum,omitempty" json:"checksum,omitempty"`
	Signature          bool              `yaml:"signature,omitempty" json:"signature,omitempty"`
	Meta               bool              `yaml:"meta,omitempty" json:"meta,omitempty"`
	CustomArtifactName bool              `yaml:"custom_artifact_name,omitempty" json:"custom_artifact_name,omitempty"`
	CustomHeaders      map[string]string `yaml:"custom_headers,omitempty" json:"custom_headers,omitempty"`
}

// Publisher configuration.
type Publisher struct {
	Name       string      `yaml:"name,omitempty" json:"name,omitempty"`
	IDs        []string    `yaml:"ids,omitempty" json:"ids,omitempty"`
	Checksum   bool        `yaml:"checksum,omitempty" json:"checksum,omitempty"`
	Signature  bool        `yaml:"signature,omitempty" json:"signature,omitempty"`
	Meta       bool        `yaml:"meta,omitempty" json:"meta,omitempty"`
	Dir        string      `yaml:"dir,omitempty" json:"dir,omitempty"`
	Cmd        string      `yaml:"cmd,omitempty" json:"cmd,omitempty"`
	Env        []string    `yaml:"env,omitempty" json:"env,omitempty"`
	ExtraFiles []ExtraFile `yaml:"extra_files,omitempty" json:"extra_files,omitempty"`
	Disable    string      `yaml:"disable,omitempty" json:"disable,omitempty" jsonschema:"oneof_type=string;boolean"`
}

// Source configuration.
type Source struct {
	NameTemplate   string `yaml:"name_template,omitempty" json:"name_template,omitempty"`
	Format         string `yaml:"format,omitempty" json:"format,omitempty" jsonschema:"enum=tar,enum=tgz,enum=tar.gz,enum=zip,default=tar.gz"`
	Enabled        bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	PrefixTemplate string `yaml:"prefix_template,omitempty" json:"prefix_template,omitempty"`
	Files          []File `yaml:"files,omitempty" json:"files,omitempty"`

	// Deprecated: don't need to set this anymore.
	RLCP string `yaml:"rlcp,omitempty" json:"rlcp,omitempty" jsonschema:"oneof_type=string;boolean,deprecated=true,description=you can now remove this"`
}

// Project includes all project configuration.
type Project struct {
	Version         int              `yaml:"version,omitempty" json:"version,omitempty" jsonschema:"enum=1,default=1"`
	ProjectName     string           `yaml:"project_name,omitempty" json:"project_name,omitempty"`
	Env             []string         `yaml:"env,omitempty" json:"env,omitempty"`
	Release         Release          `yaml:"release,omitempty" json:"release,omitempty"`
	Milestones      []Milestone      `yaml:"milestones,omitempty" json:"milestones,omitempty"`
	Brews           []Homebrew       `yaml:"brews,omitempty" json:"brews,omitempty"`
	Nix             []Nix            `yaml:"nix,omitempty" json:"nix,omitempty"`
	Winget          []Winget         `yaml:"winget,omitempty" json:"winget,omitempty"`
	AURs            []AUR            `yaml:"aurs,omitempty" json:"aurs,omitempty"`
	Krews           []Krew           `yaml:"krews,omitempty" json:"krews,omitempty"`
	Kos             []Ko             `yaml:"kos,omitempty" json:"kos,omitempty"`
	Scoops          []Scoop          `yaml:"scoops,omitempty" json:"scoops,omitempty"`
	Builds          []Build          `yaml:"builds,omitempty" json:"builds,omitempty"`
	Archives        []Archive        `yaml:"archives,omitempty" json:"archives,omitempty"`
	NFPMs           []NFPM           `yaml:"nfpms,omitempty" json:"nfpms,omitempty"`
	Snapcrafts      []Snapcraft      `yaml:"snapcrafts,omitempty" json:"snapcrafts,omitempty"`
	Snapshot        Snapshot         `yaml:"snapshot,omitempty" json:"snapshot,omitempty"`
	Checksum        Checksum         `yaml:"checksum,omitempty" json:"checksum,omitempty"`
	Dockers         []Docker         `yaml:"dockers,omitempty" json:"dockers,omitempty"`
	DockerManifests []DockerManifest `yaml:"docker_manifests,omitempty" json:"docker_manifests,omitempty"`
	Artifactories   []Upload         `yaml:"artifactories,omitempty" json:"artifactories,omitempty"`
	Uploads         []Upload         `yaml:"uploads,omitempty" json:"uploads,omitempty"`
	Blobs           []Blob           `yaml:"blobs,omitempty" json:"blobs,omitempty"`
	Publishers      []Publisher      `yaml:"publishers,omitempty" json:"publishers,omitempty"`
	Changelog       Changelog        `yaml:"changelog,omitempty" json:"changelog,omitempty"`
	Dist            string           `yaml:"dist,omitempty" json:"dist,omitempty"`
	Signs           []Sign           `yaml:"signs,omitempty" json:"signs,omitempty"`
	DockerSigns     []Sign           `yaml:"docker_signs,omitempty" json:"docker_signs,omitempty"`
	EnvFiles        EnvFiles         `yaml:"env_files,omitempty" json:"env_files,omitempty"`
	Before          Before           `yaml:"before,omitempty" json:"before,omitempty"`
	Source          Source           `yaml:"source,omitempty" json:"source,omitempty"`
	GoMod           GoMod            `yaml:"gomod,omitempty" json:"gomod,omitempty"`
	Announce        Announce         `yaml:"announce,omitempty" json:"announce,omitempty"`
	SBOMs           []SBOM           `yaml:"sboms,omitempty" json:"sboms,omitempty"`
	Chocolateys     []Chocolatey     `yaml:"chocolateys,omitempty" json:"chocolateys,omitempty"`
	Git             Git              `yaml:"git,omitempty" json:"git,omitempty"`
	ReportSizes     bool             `yaml:"report_sizes,omitempty" json:"report_sizes,omitempty"`
	Metadata        ProjectMetadata  `yaml:"metadata,omitempty" json:"metadata,omitempty"`

	UniversalBinaries []UniversalBinary `yaml:"universal_binaries,omitempty" json:"universal_binaries,omitempty"`
	UPXs              []UPX             `yaml:"upx,omitempty" json:"upx,omitempty"`

	// force the SCM token to use when multiple are set
	ForceToken string `yaml:"force_token,omitempty" json:"force_token,omitempty" jsonschema:"enum=github,enum=gitlab,enum=gitea,enum=,default="`

	// should be set if using github enterprise
	GitHubURLs GitHubURLs `yaml:"github_urls,omitempty" json:"github_urls,omitempty"`

	// should be set if using a private gitlab
	GitLabURLs GitLabURLs `yaml:"gitlab_urls,omitempty" json:"gitlab_urls,omitempty"`

	// should be set if using Gitea
	GiteaURLs GiteaURLs `yaml:"gitea_urls,omitempty" json:"gitea_urls,omitempty"`

	// Deprecated: use Scoops instead.
	Scoop Scoop `yaml:"scoop,omitempty" json:"scoop,omitempty" jsonschema:"deprecated=true,description=use scoops instead"`

	// Deprecated: use Builds instead.
	SingleBuild Build `yaml:"build,omitempty" json:"build,omitempty" jsonschema:"deprecated=true,description=use builds instead"`
}

type ProjectMetadata struct {
	ModTimestamp string `yaml:"mod_timestamp,omitempty" json:"mod_timestamp,omitempty"`
}

type GoMod struct {
	Proxy    bool     `yaml:"proxy,omitempty" json:"proxy,omitempty"`
	Env      []string `yaml:"env,omitempty" json:"env,omitempty"`
	GoBinary string   `yaml:"gobinary,omitempty" json:"gobinary,omitempty"`
	Mod      string   `yaml:"mod,omitempty" json:"mod,omitempty"`
	Dir      string   `yaml:"dir,omitempty" json:"dir,omitempty"`
}

type Announce struct {
	Skip           string         `yaml:"skip,omitempty" json:"skip,omitempty" jsonschema:"oneof_type=string;boolean"`
	Twitter        Twitter        `yaml:"twitter,omitempty" json:"twitter,omitempty"`
	Mastodon       Mastodon       `yaml:"mastodon,omitempty" json:"mastodon,omitempty"`
	Reddit         Reddit         `yaml:"reddit,omitempty" json:"reddit,omitempty"`
	Slack          Slack          `yaml:"slack,omitempty" json:"slack,omitempty"`
	Discord        Discord        `yaml:"discord,omitempty" json:"discord,omitempty"`
	Teams          Teams          `yaml:"teams,omitempty" json:"teams,omitempty"`
	SMTP           SMTP           `yaml:"smtp,omitempty" json:"smtp,omitempty"`
	Mattermost     Mattermost     `yaml:"mattermost,omitempty" json:"mattermost,omitempty"`
	LinkedIn       LinkedIn       `yaml:"linkedin,omitempty" json:"linkedin,omitempty"`
	Telegram       Telegram       `yaml:"telegram,omitempty" json:"telegram,omitempty"`
	Webhook        Webhook        `yaml:"webhook,omitempty" json:"webhook,omitempty"`
	OpenCollective OpenCollective `yaml:"opencollective,omitempty" json:"opencolletive,omitempty"`
}

type Webhook struct {
	Enabled         bool              `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	SkipTLSVerify   bool              `yaml:"skip_tls_verify,omitempty" json:"skip_tls_verify,omitempty"`
	MessageTemplate string            `yaml:"message_template,omitempty" json:"message_template,omitempty"`
	EndpointURL     string            `yaml:"endpoint_url,omitempty" json:"endpoint_url,omitempty"`
	Headers         map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	ContentType     string            `yaml:"content_type,omitempty" json:"content_type,omitempty"`
}

type Twitter struct {
	Enabled         bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	MessageTemplate string `yaml:"message_template,omitempty" json:"message_template,omitempty"`
}

type Mastodon struct {
	Enabled         bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	MessageTemplate string `yaml:"message_template,omitempty" json:"message_template,omitempty"`
	Server          string `yaml:"server" json:"server"`
}

type Reddit struct {
	Enabled       bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	ApplicationID string `yaml:"application_id,omitempty" json:"application_id,omitempty"`
	Username      string `yaml:"username,omitempty" json:"username,omitempty"`
	TitleTemplate string `yaml:"title_template,omitempty" json:"title_template,omitempty"`
	URLTemplate   string `yaml:"url_template,omitempty" json:"url_template,omitempty"`
	Sub           string `yaml:"sub,omitempty" json:"sub,omitempty"`
}

type Slack struct {
	Enabled         bool              `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	MessageTemplate string            `yaml:"message_template,omitempty" json:"message_template,omitempty"`
	Channel         string            `yaml:"channel,omitempty" json:"channel,omitempty"`
	Username        string            `yaml:"username,omitempty" json:"username,omitempty"`
	IconEmoji       string            `yaml:"icon_emoji,omitempty" json:"icon_emoji,omitempty"`
	IconURL         string            `yaml:"icon_url,omitempty" json:"icon_url,omitempty"`
	Blocks          []SlackBlock      `yaml:"blocks,omitempty" json:"blocks,omitempty"`
	Attachments     []SlackAttachment `yaml:"attachments,omitempty" json:"attachments,omitempty"`
}

type Discord struct {
	Enabled         bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	MessageTemplate string `yaml:"message_template,omitempty" json:"message_template,omitempty"`
	Author          string `yaml:"author,omitempty" json:"author,omitempty"`
	Color           string `yaml:"color,omitempty" json:"color,omitempty"`
	IconURL         string `yaml:"icon_url,omitempty" json:"icon_url,omitempty"`
}

type Teams struct {
	Enabled         bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	TitleTemplate   string `yaml:"title_template,omitempty" json:"title_template,omitempty"`
	MessageTemplate string `yaml:"message_template,omitempty" json:"message_template,omitempty"`
	Color           string `yaml:"color,omitempty" json:"color,omitempty"`
	IconURL         string `yaml:"icon_url,omitempty" json:"icon_url,omitempty"`
}

type Mattermost struct {
	Enabled         bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	MessageTemplate string `yaml:"message_template,omitempty" json:"message_template,omitempty"`
	TitleTemplate   string `yaml:"title_template,omitempty" json:"title_template,omitempty"`
	Color           string `yaml:"color,omitempty" json:"color,omitempty"`
	Channel         string `yaml:"channel,omitempty" json:"channel,omitempty"`
	Username        string `yaml:"username,omitempty" json:"username,omitempty"`
	IconEmoji       string `yaml:"icon_emoji,omitempty" json:"icon_emoji,omitempty"`
	IconURL         string `yaml:"icon_url,omitempty" json:"icon_url,omitempty"`
}

type SMTP struct {
	Enabled            bool     `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Host               string   `yaml:"host,omitempty" json:"host,omitempty"`
	Port               int      `yaml:"port,omitempty" json:"port,omitempty"`
	Username           string   `yaml:"username,omitempty" json:"username,omitempty"`
	From               string   `yaml:"from,omitempty" json:"from,omitempty"`
	To                 []string `yaml:"to,omitempty" json:"to,omitempty"`
	SubjectTemplate    string   `yaml:"subject_template,omitempty" json:"subject_template,omitempty"`
	BodyTemplate       string   `yaml:"body_template,omitempty" json:"body_template,omitempty"`
	InsecureSkipVerify bool     `yaml:"insecure_skip_verify,omitempty" json:"insecure_skip_verify,omitempty"`
}

type LinkedIn struct {
	Enabled         bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	MessageTemplate string `yaml:"message_template,omitempty" json:"message_template,omitempty"`
}

type Telegram struct {
	Enabled         bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	MessageTemplate string `yaml:"message_template,omitempty" json:"message_template,omitempty"`
	ChatID          string `yaml:"chat_id,omitempty" json:"chat_id,omitempty" jsonschema:"oneof_type=string;integer"`
	ParseMode       string `yaml:"parse_mode,omitempty" json:"parse_mode,omitempty" jsonschema:"enum=MarkdownV2,enum=HTML,default=MarkdownV2"`
}

type OpenCollective struct {
	Enabled         bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Slug            string `yaml:"slug,omitempty" json:"slug,omitempty"`
	TitleTemplate   string `yaml:"title_template,omitempty" json:"title_template,omitempty"`
	MessageTemplate string `yaml:"message_template,omitempty" json:"message_template,omitempty"`
}

// SlackBlock represents the untyped structure of a rich slack message layout.
type SlackBlock struct {
	Internal interface{}
}

// UnmarshalYAML is a custom unmarshaler that unmarshals a YAML slack block as untyped interface{}.
func (a *SlackBlock) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var yamlv2 interface{}
	if err := unmarshal(&yamlv2); err != nil {
		return err
	}

	a.Internal = yamlv2

	return nil
}

// MarshalJSON marshals a slack block as JSON.
func (a SlackBlock) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Internal)
}

// SlackAttachment represents the untyped structure of a slack message attachment.
type SlackAttachment struct {
	Internal interface{}
}

// UnmarshalYAML is a custom unmarshaler that unmarshals a YAML slack attachment as untyped interface{}.
func (a *SlackAttachment) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var yamlv2 interface{}
	if err := unmarshal(&yamlv2); err != nil {
		return err
	}

	a.Internal = yamlv2

	return nil
}

// MarshalJSON marshals a slack attachment as JSON.
func (a SlackAttachment) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Internal)
}

// Chocolatey contains the chocolatey section.
type Chocolatey struct {
	Name                     string                 `yaml:"name,omitempty" json:"name,omitempty"`
	IDs                      []string               `yaml:"ids,omitempty" json:"ids,omitempty"`
	PackageSourceURL         string                 `yaml:"package_source_url,omitempty" json:"package_source_url,omitempty"`
	Owners                   string                 `yaml:"owners,omitempty" json:"owners,omitempty"`
	Title                    string                 `yaml:"title,omitempty" json:"title,omitempty"`
	Authors                  string                 `yaml:"authors,omitempty" json:"authors,omitempty"`
	ProjectURL               string                 `yaml:"project_url,omitempty" json:"project_url,omitempty"`
	URLTemplate              string                 `yaml:"url_template,omitempty" json:"url_template,omitempty"`
	IconURL                  string                 `yaml:"icon_url,omitempty" json:"icon_url,omitempty"`
	Copyright                string                 `yaml:"copyright,omitempty" json:"copyright,omitempty"`
	LicenseURL               string                 `yaml:"license_url,omitempty" json:"license_url,omitempty"`
	RequireLicenseAcceptance bool                   `yaml:"require_license_acceptance,omitempty" json:"require_license_acceptance,omitempty"`
	ProjectSourceURL         string                 `yaml:"project_source_url,omitempty" json:"project_source_url,omitempty"`
	DocsURL                  string                 `yaml:"docs_url,omitempty" json:"docs_url,omitempty"`
	BugTrackerURL            string                 `yaml:"bug_tracker_url,omitempty" json:"bug_tracker_url,omitempty"`
	Tags                     string                 `yaml:"tags,omitempty" json:"tags,omitempty"`
	Summary                  string                 `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description              string                 `yaml:"description,omitempty" json:"description,omitempty"`
	ReleaseNotes             string                 `yaml:"release_notes,omitempty" json:"release_notes,omitempty"`
	Dependencies             []ChocolateyDependency `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	SkipPublish              bool                   `yaml:"skip_publish,omitempty" json:"skip_publish,omitempty"`
	APIKey                   string                 `yaml:"api_key,omitempty" json:"api_key,omitempty"`
	SourceRepo               string                 `yaml:"source_repo,omitempty" json:"source_repo,omitempty"`
	Goamd64                  string                 `yaml:"goamd64,omitempty" json:"goamd64,omitempty"`
}

// ChcolateyDependency represents Chocolatey dependency.
type ChocolateyDependency struct {
	ID      string `yaml:"id,omitempty" json:"id,omitempty"`
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}
