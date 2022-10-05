// Package config contains the model and loader of the goreleaser configuration
// file.
package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/yaml"
	"github.com/goreleaser/nfpm/v2/files"
	"github.com/invopop/jsonschema"
)

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
}

// HomebrewDependency represents Homebrew dependency.
type HomebrewDependency struct {
	Name    string `yaml:"name,omitempty" json:"name,omitempty"`
	Type    string `yaml:"type,omitempty" json:"type,omitempty"`
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
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
	SkipUpload            string       `yaml:"skip_upload,omitempty" json:"skip_upload,omitempty"`
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
}

// Homebrew contains the brew section.
type Homebrew struct {
	Name                  string               `yaml:"name,omitempty" json:"name,omitempty"`
	Tap                   RepoRef              `yaml:"tap,omitempty" json:"tap,omitempty"`
	CommitAuthor          CommitAuthor         `yaml:"commit_author,omitempty" json:"commit_author,omitempty"`
	CommitMessageTemplate string               `yaml:"commit_msg_template,omitempty" json:"commit_msg_template,omitempty"`
	Folder                string               `yaml:"folder,omitempty" json:"folder,omitempty"`
	Caveats               string               `yaml:"caveats,omitempty" json:"caveats,omitempty"`
	Plist                 string               `yaml:"plist,omitempty" json:"plist,omitempty"`
	Install               string               `yaml:"install,omitempty" json:"install,omitempty"`
	PostInstall           string               `yaml:"post_install,omitempty" json:"post_install,omitempty"`
	Dependencies          []HomebrewDependency `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Test                  string               `yaml:"test,omitempty" json:"test,omitempty"`
	Conflicts             []string             `yaml:"conflicts,omitempty" json:"conflicts,omitempty"`
	Description           string               `yaml:"description,omitempty" json:"description,omitempty"`
	Homepage              string               `yaml:"homepage,omitempty" json:"homepage,omitempty"`
	License               string               `yaml:"license,omitempty" json:"license,omitempty"`
	SkipUpload            string               `yaml:"skip_upload,omitempty" json:"skip_upload,omitempty"`
	DownloadStrategy      string               `yaml:"download_strategy,omitempty" json:"download_strategy,omitempty"`
	URLTemplate           string               `yaml:"url_template,omitempty" json:"url_template,omitempty"`
	CustomRequire         string               `yaml:"custom_require,omitempty" json:"custom_require,omitempty"`
	CustomBlock           string               `yaml:"custom_block,omitempty" json:"custom_block,omitempty"`
	IDs                   []string             `yaml:"ids,omitempty" json:"ids,omitempty"`
	Goarm                 string               `yaml:"goarm,omitempty" json:"goarm,omitempty"`
	Goamd64               string               `yaml:"goamd64,omitempty" json:"goamd64,omitempty"`
	Service               string               `yaml:"service,omitempty" json:"service,omitempty"`
}

// Krew contains the krew section.
type Krew struct {
	IDs                   []string     `yaml:"ids,omitempty" json:"ids,omitempty"`
	Name                  string       `yaml:"name,omitempty" json:"name,omitempty"`
	Index                 RepoRef      `yaml:"index,omitempty" json:"index,omitempty"`
	CommitAuthor          CommitAuthor `yaml:"commit_author,omitempty" json:"commit_author,omitempty"`
	CommitMessageTemplate string       `yaml:"commit_msg_template,omitempty" json:"commit_msg_template,omitempty"`
	Caveats               string       `yaml:"caveats,omitempty" json:"caveats,omitempty"`
	ShortDescription      string       `yaml:"short_description,omitempty" json:"short_description,omitempty"`
	Description           string       `yaml:"description,omitempty" json:"description,omitempty"`
	Homepage              string       `yaml:"homepage,omitempty" json:"homepage,omitempty"`
	URLTemplate           string       `yaml:"url_template,omitempty" json:"url_template,omitempty"`
	Goarm                 string       `yaml:"goarm,omitempty" json:"goarm,omitempty"`
	Goamd64               string       `yaml:"goamd64,omitempty" json:"goamd64,omitempty"`
	SkipUpload            string       `yaml:"skip_upload,omitempty" json:"skip_upload,omitempty"`
}

// Scoop contains the scoop.sh section.
type Scoop struct {
	Name                  string       `yaml:"name,omitempty" json:"name,omitempty"`
	Bucket                RepoRef      `yaml:"bucket,omitempty" json:"bucket,omitempty"`
	Folder                string       `yaml:"folder,omitempty" json:"folder,omitempty"`
	CommitAuthor          CommitAuthor `yaml:"commit_author,omitempty" json:"commit_author,omitempty"`
	CommitMessageTemplate string       `yaml:"commit_msg_template,omitempty" json:"commit_msg_template,omitempty"`
	Homepage              string       `yaml:"homepage,omitempty" json:"homepage,omitempty"`
	Description           string       `yaml:"description,omitempty" json:"description,omitempty"`
	License               string       `yaml:"license,omitempty" json:"license,omitempty"`
	URLTemplate           string       `yaml:"url_template,omitempty" json:"url_template,omitempty"`
	Persist               []string     `yaml:"persist,omitempty" json:"persist,omitempty"`
	SkipUpload            string       `yaml:"skip_upload,omitempty" json:"skip_upload,omitempty"`
	PreInstall            []string     `yaml:"pre_install,omitempty" json:"pre_install,omitempty"`
	PostInstall           []string     `yaml:"post_install,omitempty" json:"post_install,omitempty"`
	Goamd64               string       `yaml:"goamd64,omitempty" json:"goamd64,omitempty"`
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
	Goarm   string `yaml:"goarm,omitempty" json:"goarm,omitempty"`
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
	Goarm        string                          `yaml:"goarm,omitempty" json:"goarm,omitempty"`
	Gomips       string                          `yaml:"gomips,omitempty" json:"gomips,omitempty"`
	Goamd64      string                          `yaml:"goamd64,omitempty" json:"goamd64,omitempty"`
	BuildDetails `yaml:",inline" json:",inline"` // nolint: tagliatelle
}

type BuildDetails struct {
	Ldflags  StringArray `yaml:"ldflags,omitempty" json:"ldflags,omitempty"`
	Tags     FlagArray   `yaml:"tags,omitempty" json:"tags,omitempty"`
	Flags    FlagArray   `yaml:"flags,omitempty" json:"flags,omitempty"`
	Asmflags StringArray `yaml:"asmflags,omitempty" json:"asmflags,omitempty"`
	Gcflags  StringArray `yaml:"gcflags,omitempty" json:"gcflags,omitempty"`
	Env      []string    `yaml:"env,omitempty" json:"env,omitempty"`
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
	type t Hook
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	schema := reflector.Reflect(&t{})
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
	Format string `yaml:"format,omitempty" json:"format,omitempty"`
}

// File is a file inside an archive.
type File struct {
	Source      string   `yaml:"src,omitempty" json:"src,omitempty"`
	Destination string   `yaml:"dst,omitempty" json:"dst,omitempty"`
	StripParent bool     `yaml:"strip_parent,omitempty" json:"strip_parent,omitempty"`
	Info        FileInfo `yaml:"info,omitempty" json:"info,omitempty"`
}

// FileInfo is the file info of a file.
type FileInfo struct {
	Owner string      `yaml:"owner,omitempty" json:"owner,omitempty"`
	Group string      `yaml:"group,omitempty" json:"group,omitempty"`
	Mode  os.FileMode `yaml:"mode,omitempty" json:"mode,omitempty"`
	MTime time.Time   `yaml:"mtime,omitempty" json:"mtime,omitempty"`
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
	type t File
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	schema := reflector.Reflect(&t{})
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
	ID           string          `yaml:"id,omitempty" json:"id,omitempty"` // deprecated
	IDs          []string        `yaml:"ids,omitempty" json:"ids,omitempty"`
	NameTemplate string          `yaml:"name_template,omitempty" json:"name_template,omitempty"`
	Replace      bool            `yaml:"replace,omitempty" json:"replace,omitempty"`
	Hooks        BuildHookConfig `yaml:"hooks,omitempty" json:"hooks,omitempty"`
}

// Archive config used for the archive.
type Archive struct {
	ID                        string            `yaml:"id,omitempty" json:"id,omitempty"`
	Builds                    []string          `yaml:"builds,omitempty" json:"builds,omitempty"`
	NameTemplate              string            `yaml:"name_template,omitempty" json:"name_template,omitempty"`
	Replacements              map[string]string `yaml:"replacements,omitempty" json:"replacements,omitempty"`
	Format                    string            `yaml:"format,omitempty" json:"format,omitempty"`
	FormatOverrides           []FormatOverride  `yaml:"format_overrides,omitempty" json:"format_overrides,omitempty"`
	WrapInDirectory           string            `yaml:"wrap_in_directory,omitempty" json:"wrap_in_directory,omitempty" jsonschema:"oneof_type=string;boolean"`
	StripParentBinaryFolder   bool              `yaml:"strip_parent_binary_folder,omitempty" json:"strip_parent_binary_folder,omitempty"`
	Files                     []File            `yaml:"files,omitempty" json:"files,omitempty"`
	Meta                      bool              `yaml:"meta,omitempty" json:"meta,omitempty"`
	AllowDifferentBinaryCount bool              `yaml:"allow_different_binary_count,omitempty" json:"allow_different_binary_count,omitempty"`
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
	Disable                bool        `yaml:"disable,omitempty" json:"disable,omitempty"`
	SkipUpload             bool        `yaml:"skip_upload,omitempty" json:"skip_upload,omitempty"`
	Prerelease             string      `yaml:"prerelease,omitempty" json:"prerelease,omitempty"`
	NameTemplate           string      `yaml:"name_template,omitempty" json:"name_template,omitempty"`
	IDs                    []string    `yaml:"ids,omitempty" json:"ids,omitempty"`
	ExtraFiles             []ExtraFile `yaml:"extra_files,omitempty" json:"extra_files,omitempty"`
	DiscussionCategoryName string      `yaml:"discussion_category_name,omitempty" json:"discussion_category_name,omitempty"`
	Header                 string      `yaml:"header,omitempty" json:"header,omitempty"`
	Footer                 string      `yaml:"footer,omitempty" json:"footer,omitempty"`

	ReleaseNotesMode ReleaseNotesMode `yaml:"mode,omitempty" json:"mode,omitempty" jsonschema:"enum=keep-existing,enum=append,enum=prepend,enum=replace,default=keep-existing"`
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
	Formats     []string `yaml:"formats,omitempty" json:"formats,omitempty"`
	Section     string   `yaml:"section,omitempty" json:"section,omitempty"`
	Priority    string   `yaml:"priority,omitempty" json:"priority,omitempty"`
	Vendor      string   `yaml:"vendor,omitempty" json:"vendor,omitempty"`
	Homepage    string   `yaml:"homepage,omitempty" json:"homepage,omitempty"`
	Maintainer  string   `yaml:"maintainer,omitempty" json:"maintainer,omitempty"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	License     string   `yaml:"license,omitempty" json:"license,omitempty"`
	Bindir      string   `yaml:"bindir,omitempty" json:"bindir,omitempty"`
	Changelog   string   `yaml:"changelog,omitempty" json:"changelog,omitempty"`
	Meta        bool     `yaml:"meta,omitempty" json:"meta,omitempty"` // make package without binaries - only deps
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
	Scripts   NFPMDebScripts   `yaml:"scripts,omitempty" json:"scripts,omitempty"`
	Triggers  NFPMDebTriggers  `yaml:"triggers,omitempty" json:"triggers,omitempty"`
	Breaks    []string         `yaml:"breaks,omitempty" json:"breaks,omitempty"`
	Signature NFPMDebSignature `yaml:"signature,omitempty" json:"signature,omitempty"`
	Lintian   []string         `yaml:"lintian_overrides,omitempty" json:"lintian_overrides,omitempty"`
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

// NFPMOverridables is used to specify per package format settings.
type NFPMOverridables struct {
	FileNameTemplate string            `yaml:"file_name_template,omitempty" json:"file_name_template,omitempty"`
	PackageName      string            `yaml:"package_name,omitempty" json:"package_name,omitempty"`
	Epoch            string            `yaml:"epoch,omitempty" json:"epoch,omitempty"`
	Release          string            `yaml:"release,omitempty" json:"release,omitempty"`
	Prerelease       string            `yaml:"prerelease,omitempty" json:"prerelease,omitempty"`
	VersionMetadata  string            `yaml:"version_metadata,omitempty" json:"version_metadata,omitempty"`
	Replacements     map[string]string `yaml:"replacements,omitempty" json:"replacements,omitempty"`
	Dependencies     []string          `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Recommends       []string          `yaml:"recommends,omitempty" json:"recommends,omitempty"`
	Suggests         []string          `yaml:"suggests,omitempty" json:"suggests,omitempty"`
	Conflicts        []string          `yaml:"conflicts,omitempty" json:"conflicts,omitempty"`
	Replaces         []string          `yaml:"replaces,omitempty" json:"replaces,omitempty"`
	Provides         []string          `yaml:"provides,omitempty" json:"provides,omitempty"`
	Contents         files.Contents    `yaml:"contents,omitempty" json:"contents,omitempty"`
	Scripts          NFPMScripts       `yaml:"scripts,omitempty" json:"scripts,omitempty"`
	RPM              NFPMRPM           `yaml:"rpm,omitempty" json:"rpm,omitempty"`
	Deb              NFPMDeb           `yaml:"deb,omitempty" json:"deb,omitempty"`
	APK              NFPMAPK           `yaml:"apk,omitempty" json:"apk,omitempty"`
}

// SBOM config.
type SBOM struct {
	ID        string   `yaml:"id,omitempty" json:"id,omitempty"`
	Cmd       string   `yaml:"cmd,omitempty" json:"cmd,omitempty"`
	Env       []string `yaml:"env,omitempty" json:"env,omitempty"`
	Args      []string `yaml:"args,omitempty" json:"args,omitempty"`
	Documents []string `yaml:"documents,omitempty" json:"documents,omitempty"`
	Artifacts string   `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
	IDs       []string `yaml:"ids,omitempty" json:"ids,omitempty"`
}

// Sign config.
type Sign struct {
	ID          string   `yaml:"id,omitempty" json:"id,omitempty"`
	Cmd         string   `yaml:"cmd,omitempty" json:"cmd,omitempty"`
	Args        []string `yaml:"args,omitempty" json:"args,omitempty"`
	Signature   string   `yaml:"signature,omitempty" json:"signature,omitempty"`
	Artifacts   string   `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
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
	NameTemplate string            `yaml:"name_template,omitempty" json:"name_template,omitempty"`
	Replacements map[string]string `yaml:"replacements,omitempty" json:"replacements,omitempty"`
	Publish      bool              `yaml:"publish,omitempty" json:"publish,omitempty"`

	ID               string                             `yaml:"id,omitempty" json:"id,omitempty"`
	Builds           []string                           `yaml:"builds,omitempty" json:"builds,omitempty"`
	Name             string                             `yaml:"name,omitempty" json:"name,omitempty"`
	Summary          string                             `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description      string                             `yaml:"description,omitempty" json:"description,omitempty"`
	Base             string                             `yaml:"base,omitempty" json:"base,omitempty"`
	License          string                             `yaml:"license,omitempty" json:"license,omitempty"`
	Grade            string                             `yaml:"grade,omitempty" json:"grade,omitempty"`
	ChannelTemplates []string                           `yaml:"channel_templates,omitempty" json:"channel_templates,omitempty"`
	Confinement      string                             `yaml:"confinement,omitempty" json:"confinement,omitempty"`
	Layout           map[string]SnapcraftLayoutMetadata `yaml:"layout,omitempty" json:"layout,omitempty"`
	Apps             map[string]SnapcraftAppMetadata    `yaml:"apps,omitempty" json:"apps,omitempty"`
	Plugs            map[string]interface{}             `yaml:"plugs,omitempty" json:"plugs,omitempty"`

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
	Goarm              string   `yaml:"goarm,omitempty" json:"goarm,omitempty"`
	Goamd64            string   `yaml:"goamd64,omitempty" json:"goamd64,omitempty"`
	Dockerfile         string   `yaml:"dockerfile,omitempty" json:"dockerfile,omitempty"`
	ImageTemplates     []string `yaml:"image_templates,omitempty" json:"image_templates,omitempty"`
	SkipPush           string   `yaml:"skip_push,omitempty" json:"skip_push,omitempty"`
	Files              []string `yaml:"extra_files,omitempty" json:"extra_files,omitempty"`
	BuildFlagTemplates []string `yaml:"build_flag_templates,omitempty" json:"build_flag_templates,omitempty"`
	PushFlags          []string `yaml:"push_flags,omitempty" json:"push_flags,omitempty"`
	Use                string   `yaml:"use,omitempty" json:"use,omitempty"`
}

// DockerManifest config.
type DockerManifest struct {
	ID             string   `yaml:"id,omitempty" json:"id,omitempty"`
	NameTemplate   string   `yaml:"name_template,omitempty" json:"name_template,omitempty"`
	SkipPush       string   `yaml:"skip_push,omitempty" json:"skip_push,omitempty"`
	ImageTemplates []string `yaml:"image_templates,omitempty" json:"image_templates,omitempty"`
	CreateFlags    []string `yaml:"create_flags,omitempty" json:"create_flags,omitempty"`
	PushFlags      []string `yaml:"push_flags,omitempty" json:"push_flags,omitempty"`
	Use            string   `yaml:"use,omitempty" json:"use,omitempty"`
}

// Filters config.
type Filters struct {
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

// Changelog Config.
type Changelog struct {
	Filters Filters          `yaml:"filters,omitempty" json:"filters,omitempty"`
	Sort    string           `yaml:"sort,omitempty" json:"sort,omitempty"`
	Skip    bool             `yaml:"skip,omitempty" json:"skip,omitempty"` // TODO(caarlos0): rename to Disable to match other pipes
	Use     string           `yaml:"use,omitempty" json:"use,omitempty" jsonschema:"enum=git,enum=github,enum=github-native,enum=gitlab,default=git"`
	Groups  []ChangeLogGroup `yaml:"groups,omitempty" json:"groups,omitempty"`
	Abbrev  int              `yaml:"abbrev,omitempty" json:"abbrev,omitempty"`
}

// ChangeLogGroup holds the grouping criteria for the changelog.
type ChangeLogGroup struct {
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
	Bucket     string      `yaml:"bucket,omitempty" json:"bucket,omitempty"`
	Provider   string      `yaml:"provider,omitempty" json:"provider,omitempty"`
	Region     string      `yaml:"region,omitempty" json:"region,omitempty"`
	DisableSSL bool        `yaml:"disableSSL,omitempty" json:"disableSSL,omitempty"` // nolint:tagliatelle // TODO(caarlos0): rename to disable_ssl
	Folder     string      `yaml:"folder,omitempty" json:"folder,omitempty"`
	KMSKey     string      `yaml:"kmskey,omitempty" json:"kmskey,omitempty"`
	IDs        []string    `yaml:"ids,omitempty" json:"ids,omitempty"`
	Endpoint   string      `yaml:"endpoint,omitempty" json:"endpoint,omitempty"` // used for minio for example
	ExtraFiles []ExtraFile `yaml:"extra_files,omitempty" json:"extra_files,omitempty"`
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
	CustomArtifactName bool              `yaml:"custom_artifact_name,omitempty" json:"custom_artifact_name,omitempty"`
	CustomHeaders      map[string]string `yaml:"custom_headers,omitempty" json:"custom_headers,omitempty"`
}

// Publisher configuration.
type Publisher struct {
	Name       string      `yaml:"name,omitempty" json:"name,omitempty"`
	IDs        []string    `yaml:"ids,omitempty" json:"ids,omitempty"`
	Checksum   bool        `yaml:"checksum,omitempty" json:"checksum,omitempty"`
	Signature  bool        `yaml:"signature,omitempty" json:"signature,omitempty"`
	Dir        string      `yaml:"dir,omitempty" json:"dir,omitempty"`
	Cmd        string      `yaml:"cmd,omitempty" json:"cmd,omitempty"`
	Env        []string    `yaml:"env,omitempty" json:"env,omitempty"`
	ExtraFiles []ExtraFile `yaml:"extra_files,omitempty" json:"extra_files,omitempty"`
}

// Source configuration.
type Source struct {
	NameTemplate   string `yaml:"name_template,omitempty" json:"name_template,omitempty"`
	Format         string `yaml:"format,omitempty" json:"format,omitempty"`
	Enabled        bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	PrefixTemplate string `yaml:"prefix_template,omitempty" json:"prefix_template,omitempty"`
	Files          []File `yaml:"files,omitempty" json:"files,omitempty"`
}

// Project includes all project configuration.
type Project struct {
	ProjectName     string           `yaml:"project_name,omitempty" json:"project_name,omitempty"`
	Env             []string         `yaml:"env,omitempty" json:"env,omitempty"`
	Release         Release          `yaml:"release,omitempty" json:"release,omitempty"`
	Milestones      []Milestone      `yaml:"milestones,omitempty" json:"milestones,omitempty"`
	Brews           []Homebrew       `yaml:"brews,omitempty" json:"brews,omitempty"`
	AURs            []AUR            `yaml:"aurs,omitempty" json:"aurs,omitempty"`
	Krews           []Krew           `yaml:"krews,omitempty" json:"krews,omitempty"`
	Scoop           Scoop            `yaml:"scoop,omitempty" json:"scoop,omitempty"`
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

	UniversalBinaries []UniversalBinary `yaml:"universal_binaries,omitempty" json:"universal_binaries,omitempty"`

	// this is a hack ¯\_(ツ)_/¯
	SingleBuild Build `yaml:"build,omitempty" json:"build,omitempty"`

	// should be set if using github enterprise
	GitHubURLs GitHubURLs `yaml:"github_urls,omitempty" json:"github_urls,omitempty"`

	// should be set if using a private gitlab
	GitLabURLs GitLabURLs `yaml:"gitlab_urls,omitempty" json:"gitlab_urls,omitempty"`

	// should be set if using Gitea
	GiteaURLs GiteaURLs `yaml:"gitea_urls,omitempty" json:"gitea_urls,omitempty"`
}

type GoMod struct {
	Proxy    bool     `yaml:"proxy,omitempty" json:"proxy,omitempty"`
	Env      []string `yaml:"env,omitempty" json:"env,omitempty"`
	GoBinary string   `yaml:"gobinary,omitempty" json:"gobinary,omitempty"`
	Mod      string   `yaml:"mod,omitempty" json:"mod,omitempty"`
}

type Announce struct {
	Skip       string     `yaml:"skip,omitempty" json:"skip,omitempty"`
	Twitter    Twitter    `yaml:"twitter,omitempty" json:"twitter,omitempty"`
	Reddit     Reddit     `yaml:"reddit,omitempty" json:"reddit,omitempty"`
	Slack      Slack      `yaml:"slack,omitempty" json:"slack,omitempty"`
	Discord    Discord    `yaml:"discord,omitempty" json:"discord,omitempty"`
	Teams      Teams      `yaml:"teams,omitempty" json:"teams,omitempty"`
	SMTP       SMTP       `yaml:"smtp,omitempty" json:"smtp,omitempty"`
	Mattermost Mattermost `yaml:"mattermost,omitempty" json:"mattermost,omitempty"`
	LinkedIn   LinkedIn   `yaml:"linkedin,omitempty" json:"linkedin,omitempty"`
	Telegram   Telegram   `yaml:"telegram,omitempty" json:"telegram,omitempty"`
	Webhook    Webhook    `yaml:"webhook,omitempty" json:"webhook,omitempty"`
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
	ChatID          int64  `yaml:"chat_id,omitempty" json:"chat_id,omitempty"`
}

// Load config file.
func Load(file string) (config Project, err error) {
	f, err := os.Open(file) // #nosec
	if err != nil {
		return
	}
	defer f.Close()
	log.WithField("file", file).Info("loading config file")
	return LoadReader(f)
}

// LoadReader config via io.Reader.
func LoadReader(fd io.Reader) (config Project, err error) {
	data, err := io.ReadAll(fd)
	if err != nil {
		return config, err
	}
	err = yaml.UnmarshalStrict(data, &config)
	log.WithField("config", config).Debug("loaded config file")
	return config, err
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
