// Package config contains the model and loader of the goreleaser configuration
// file.
package config

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/jsonschema"

	"github.com/apex/log"
	"github.com/goreleaser/nfpm/v2/files"
	yaml "gopkg.in/yaml.v2"
)

// GitHubURLs holds the URLs to be used when using github enterprise.
type GitHubURLs struct {
	API           string `yaml:"api,omitempty"`
	Upload        string `yaml:"upload,omitempty"`
	Download      string `yaml:"download,omitempty"`
	SkipTLSVerify bool   `yaml:"skip_tls_verify,omitempty"`
}

// GitLabURLs holds the URLs to be used when using gitlab ce/enterprise.
type GitLabURLs struct {
	API                string `yaml:"api,omitempty"`
	Download           string `yaml:"download,omitempty"`
	SkipTLSVerify      bool   `yaml:"skip_tls_verify,omitempty"`
	UsePackageRegistry bool   `yaml:"use_package_registry,omitempty"`
}

// GiteaURLs holds the URLs to be used when using gitea.
type GiteaURLs struct {
	API           string `yaml:"api,omitempty"`
	Download      string `yaml:"download,omitempty"`
	SkipTLSVerify bool   `yaml:"skip_tls_verify,omitempty"`
}

// Repo represents any kind of repo (github, gitlab, etc).
// to upload releases into.
type Repo struct {
	Owner string `yaml:"owner,omitempty"`
	Name  string `yaml:"name,omitempty"`
}

// String of the repo, e.g. owner/name.
func (r Repo) String() string {
	if r.Owner == "" && r.Name == "" {
		return ""
	}
	return r.Owner + "/" + r.Name
}

// RepoRef represents any kind of repo which may differ
// from the one we are building from and may therefore
// also require separate authentication
// e.g. Homebrew Tap, Scoop bucket.
type RepoRef struct {
	Owner  string `yaml:"owner,omitempty"`
	Name   string `yaml:"name,omitempty"`
	Token  string `yaml:"token,omitempty"`
	Branch string `yaml:"branch,omitempty"`
}

// HomebrewDependency represents Homebrew dependency.
type HomebrewDependency struct {
	Name string `yaml:"name,omitempty"`
	Type string `yaml:"type,omitempty"`
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

func (a HomebrewDependency) JSONSchemaType() *jsonschema.Type {
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	schema := reflector.Reflect(&homebrewDependency{})
	return &jsonschema.Type{
		OneOf: []*jsonschema.Type{
			{
				Type: "string",
			},
			schema.Type,
		},
	}
}

type AUR struct {
	Name                  string       `yaml:"name,omitempty"`
	IDs                   []string     `yaml:"ids,omitempty"`
	CommitAuthor          CommitAuthor `yaml:"commit_author,omitempty"`
	CommitMessageTemplate string       `yaml:"commit_msg_template,omitempty"`
	Description           string       `yaml:"description,omitempty"`
	Homepage              string       `yaml:"homepage,omitempty"`
	License               string       `yaml:"license,omitempty"`
	SkipUpload            string       `yaml:"skip_upload,omitempty"`
	URLTemplate           string       `yaml:"url_template,omitempty"`
	Maintainers           []string     `yaml:"maintainers,omitempty"`
	Contributors          []string     `yaml:"contributors,omitempty"`
	Provides              []string     `yaml:"provides,omitempty"`
	Conflicts             []string     `yaml:"conflicts,omitempty"`
	Depends               []string     `yaml:"depends,omitempty"`
	OptDepends            []string     `yaml:"optdepends,omitempty"`
	Rel                   string       `yaml:"rel,omitempty"`
	Package               string       `yaml:"package,omitempty"`
	GitURL                string       `yaml:"git_url,omitempty"`
	GitSSHCommand         string       `yaml:"git_ssh_command,omitempty"`
	PrivateKey            string       `yaml:"private_key,omitempty"`
}

// GoFish contains the gofish section.
type GoFish struct {
	Name                  string       `yaml:"name,omitempty"`
	Rig                   RepoRef      `yaml:"rig,omitempty"`
	CommitAuthor          CommitAuthor `yaml:"commit_author,omitempty"`
	CommitMessageTemplate string       `yaml:"commit_msg_template,omitempty"`
	Description           string       `yaml:"description,omitempty"`
	Homepage              string       `yaml:"homepage,omitempty"`
	License               string       `yaml:"license,omitempty"`
	SkipUpload            string       `yaml:"skip_upload,omitempty"`
	URLTemplate           string       `yaml:"url_template,omitempty"`
	IDs                   []string     `yaml:"ids,omitempty"`
	Goarm                 string       `yaml:"goarm,omitempty"`
}

// Homebrew contains the brew section.
type Homebrew struct {
	Name                  string               `yaml:"name,omitempty"`
	Tap                   RepoRef              `yaml:"tap,omitempty"`
	CommitAuthor          CommitAuthor         `yaml:"commit_author,omitempty"`
	CommitMessageTemplate string               `yaml:"commit_msg_template,omitempty"`
	Folder                string               `yaml:"folder,omitempty"`
	Caveats               string               `yaml:"caveats,omitempty"`
	Plist                 string               `yaml:"plist,omitempty"`
	Install               string               `yaml:"install,omitempty"`
	PostInstall           string               `yaml:"post_install,omitempty"`
	Dependencies          []HomebrewDependency `yaml:"dependencies,omitempty"`
	Test                  string               `yaml:"test,omitempty"`
	Conflicts             []string             `yaml:"conflicts,omitempty"`
	Description           string               `yaml:"description,omitempty"`
	Homepage              string               `yaml:"homepage,omitempty"`
	License               string               `yaml:"license,omitempty"`
	SkipUpload            string               `yaml:"skip_upload,omitempty"`
	DownloadStrategy      string               `yaml:"download_strategy,omitempty"`
	URLTemplate           string               `yaml:"url_template,omitempty"`
	CustomRequire         string               `yaml:"custom_require,omitempty"`
	CustomBlock           string               `yaml:"custom_block,omitempty"`
	IDs                   []string             `yaml:"ids,omitempty"`
	Goarm                 string               `yaml:"goarm,omitempty"`
}

// Krew contains the krew section.
type Krew struct {
	IDs                   []string     `yaml:"ids,omitempty"`
	Name                  string       `yaml:"name,omitempty"`
	Index                 RepoRef      `yaml:"index,omitempty"`
	CommitAuthor          CommitAuthor `yaml:"commit_author,omitempty"`
	CommitMessageTemplate string       `yaml:"commit_msg_template,omitempty"`
	Caveats               string       `yaml:"caveats,omitempty"`
	ShortDescription      string       `yaml:"short_description,omitempty"`
	Description           string       `yaml:"description,omitempty"`
	Homepage              string       `yaml:"homepage,omitempty"`
	URLTemplate           string       `yaml:"url_template,omitempty"`
	Goarm                 string       `yaml:"goarm,omitempty"`
	SkipUpload            string       `yaml:"skip_upload,omitempty"`
}

// Scoop contains the scoop.sh section.
type Scoop struct {
	Name                  string       `yaml:"name,omitempty"`
	Bucket                RepoRef      `yaml:"bucket,omitempty"`
	Folder                string       `yaml:"folder,omitempty"`
	CommitAuthor          CommitAuthor `yaml:"commit_author,omitempty"`
	CommitMessageTemplate string       `yaml:"commit_msg_template,omitempty"`
	Homepage              string       `yaml:"homepage,omitempty"`
	Description           string       `yaml:"description,omitempty"`
	License               string       `yaml:"license,omitempty"`
	URLTemplate           string       `yaml:"url_template,omitempty"`
	Persist               []string     `yaml:"persist,omitempty"`
	SkipUpload            string       `yaml:"skip_upload,omitempty"`
	PreInstall            []string     `yaml:"pre_install,omitempty"`
	PostInstall           []string     `yaml:"post_install,omitempty"`
}

// CommitAuthor is the author of a Git commit.
type CommitAuthor struct {
	Name  string `yaml:"name,omitempty"`
	Email string `yaml:"email,omitempty"`
}

// BuildHooks define actions to run before and/or after something.
type BuildHooks struct { // renamed on pro
	Pre  string `yaml:"pre,omitempty"`
	Post string `yaml:"post,omitempty"`
}

// IgnoredBuild represents a build ignored by the user.
type IgnoredBuild struct {
	Goos   string `yaml:"goos,omitempty"`
	Goarch string `yaml:"goarch,omitempty"`
	Goarm  string `yaml:"goarm,omitempty"`
	Gomips string `yaml:"gomips,omitempty"`
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

func (a StringArray) JSONSchemaType() *jsonschema.Type {
	return &jsonschema.Type{
		OneOf: []*jsonschema.Type{{
			Type: "string",
		}, {
			Type: "array",
			Items: &jsonschema.Type{
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

func (a FlagArray) JSONSchemaType() *jsonschema.Type {
	return &jsonschema.Type{
		OneOf: []*jsonschema.Type{{
			Type: "string",
		}, {
			Type: "array",
			Items: &jsonschema.Type{
				Type: "string",
			},
		}},
	}
}

// Build contains the build configuration section.
type Build struct {
	ID              string          `yaml:"id,omitempty"`
	Goos            []string        `yaml:"goos,omitempty"`
	Goarch          []string        `yaml:"goarch,omitempty"`
	Goarm           []string        `yaml:"goarm,omitempty"`
	Gomips          []string        `yaml:"gomips,omitempty"`
	Targets         []string        `yaml:"targets,omitempty"`
	Ignore          []IgnoredBuild  `yaml:"ignore,omitempty"`
	Dir             string          `yaml:"dir,omitempty"`
	Main            string          `yaml:"main,omitempty"`
	Binary          string          `yaml:"binary,omitempty"`
	Hooks           BuildHookConfig `yaml:"hooks,omitempty"`
	Env             []string        `yaml:"env,omitempty"`
	Builder         string          `yaml:"builder,omitempty"`
	ModTimestamp    string          `yaml:"mod_timestamp,omitempty"`
	Skip            bool            `yaml:"skip,omitempty"`
	GoBinary        string          `yaml:"gobinary,omitempty"`
	NoUniqueDistDir bool            `yaml:"no_unique_dist_dir,omitempty"`
	UnproxiedMain   string          `yaml:"-"` // used by gomod.proxy
	UnproxiedDir    string          `yaml:"-"` // used by gomod.proxy

	BuildDetails          `yaml:",inline"`       // nolint: tagliatelle
	BuildDetailsOverrides []BuildDetailsOverride `yaml:"overrides,omitempty"`
}

type BuildDetailsOverride struct {
	Goos         string           `yaml:"goos,omitempty"`
	Goarch       string           `yaml:"goarch,omitempty"`
	Goarm        string           `yaml:"goarm,omitempty"`
	Gomips       string           `yaml:"gomips,omitempty"`
	BuildDetails `yaml:",inline"` // nolint: tagliatelle
}

type BuildDetails struct {
	Ldflags  StringArray `yaml:"ldflags,omitempty"`
	Tags     FlagArray   `yaml:"tags,omitempty"`
	Flags    FlagArray   `yaml:"flags,omitempty"`
	Asmflags StringArray `yaml:"asmflags,omitempty"`
	Gcflags  StringArray `yaml:"gcflags,omitempty"`
}

type BuildHookConfig struct {
	Pre  Hooks `yaml:"pre,omitempty"`
	Post Hooks `yaml:"post,omitempty"`
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

func (bhc Hooks) JSONSchemaType() *jsonschema.Type {
	type t Hooks
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	schema := reflector.Reflect(&t{})
	return &jsonschema.Type{
		Items: &jsonschema.Type{
			OneOf: []*jsonschema.Type{
				{
					Type: "string",
				},
				schema.Type,
			},
		},
	}
}

type Hook struct {
	Dir    string   `yaml:"dir,omitempty"`
	Cmd    string   `yaml:"cmd,omitempty"`
	Env    []string `yaml:"env,omitempty"`
	Output bool     `yaml:"output,omitempty"`
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

func (bh Hook) JSONSchemaType() *jsonschema.Type {
	type t Hook
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	schema := reflector.Reflect(&t{})
	return &jsonschema.Type{
		OneOf: []*jsonschema.Type{
			{
				Type: "string",
			},
			schema.Type,
		},
	}
}

// FormatOverride is used to specify a custom format for a specific GOOS.
type FormatOverride struct {
	Goos   string `yaml:"goos,omitempty"`
	Format string `yaml:"format,omitempty"`
}

// File is a file inside an archive.
type File struct {
	Source      string   `yaml:"src,omitempty"`
	Destination string   `yaml:"dst,omitempty"`
	StripParent bool     `yaml:"strip_parent,omitempty"`
	Info        FileInfo `yaml:"info,omitempty"`
}

// FileInfo is the file info of a file.
type FileInfo struct {
	Owner string      `yaml:"owner,omitempty"`
	Group string      `yaml:"group,omitempty"`
	Mode  os.FileMode `yaml:"mode,omitempty"`
	MTime time.Time   `yaml:"mtime,omitempty"`
}

// type alias to prevent stack overflow
type fileAlias File

// UnmarshalYAML is a custom unmarshaler that wraps strings in arrays.
func (f *File) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err == nil {
		*f = File{Source: str}
		return nil
	}

	var file fileAlias
	if err := unmarshal(&file); err != nil {
		return err
	}
	*f = File(file)
	return nil
}

func (f File) JSONSchemaType() *jsonschema.Type {
	type t File
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
	}
	schema := reflector.Reflect(&t{})
	// jsonschema would just refer to FileInfo in the definition. It doesn't get included there, as we override the
	// generated schema with JSONSchemaType here. So we need to include it directly in the schema of File.
	schema.Properties.Set("info", reflector.Reflect(&FileInfo{}).Type)
	return &jsonschema.Type{
		OneOf: []*jsonschema.Type{
			{
				Type: "string",
			},
			schema.Type,
		},
	}
}

// UniversalBinary setups macos universal binaries.
type UniversalBinary struct {
	ID           string          `yaml:"id,omitempty"` // deprecated
	IDs          []string        `yaml:"ids,omitempty"`
	NameTemplate string          `yaml:"name_template,omitempty"`
	Replace      bool            `yaml:"replace,omitempty"`
	Hooks        BuildHookConfig `yaml:"hooks,omitempty"`
}

// Archive config used for the archive.
type Archive struct {
	ID                        string            `yaml:"id,omitempty"`
	Builds                    []string          `yaml:"builds,omitempty"`
	NameTemplate              string            `yaml:"name_template,omitempty"`
	Replacements              map[string]string `yaml:"replacements,omitempty"`
	Format                    string            `yaml:"format,omitempty"`
	FormatOverrides           []FormatOverride  `yaml:"format_overrides,omitempty"`
	WrapInDirectory           string            `yaml:"wrap_in_directory,omitempty"`
	Files                     []File            `yaml:"files,omitempty"`
	AllowDifferentBinaryCount bool              `yaml:"allow_different_binary_count,omitempty"`
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
	GitHub                 Repo        `yaml:"github,omitempty"`
	GitLab                 Repo        `yaml:"gitlab,omitempty"`
	Gitea                  Repo        `yaml:"gitea,omitempty"`
	Draft                  bool        `yaml:"draft,omitempty"`
	Disable                bool        `yaml:"disable,omitempty"`
	Prerelease             string      `yaml:"prerelease,omitempty"`
	NameTemplate           string      `yaml:"name_template,omitempty"`
	IDs                    []string    `yaml:"ids,omitempty"`
	ExtraFiles             []ExtraFile `yaml:"extra_files,omitempty"`
	DiscussionCategoryName string      `yaml:"discussion_category_name,omitempty"`
	Header                 string      `yaml:"header,omitempty"`
	Footer                 string      `yaml:"footer,omitempty"`

	ReleaseNotesMode ReleaseNotesMode `yaml:"mode,omitempty" jsonschema:"enum=keep-existing,enum=append,enum=prepend,enum=replace,default=keep-existing"`
}

// Milestone config used for VCS milestone.
type Milestone struct {
	Repo         Repo   `yaml:"repo,omitempty"`
	Close        bool   `yaml:"close,omitempty"`
	FailOnError  bool   `yaml:"fail_on_error,omitempty"`
	NameTemplate string `yaml:"name_template,omitempty"`
}

// ExtraFile on a release.
type ExtraFile struct {
	Glob         string `yaml:"glob,omitempty"`
	NameTemplate string `yaml:"name_template,omitempty"`
}

// NFPM config.
type NFPM struct {
	NFPMOverridables `yaml:",inline"`            // nolint: tagliatelle
	Overrides        map[string]NFPMOverridables `yaml:"overrides,omitempty"`

	ID          string   `yaml:"id,omitempty"`
	Builds      []string `yaml:"builds,omitempty"`
	Formats     []string `yaml:"formats,omitempty"`
	Section     string   `yaml:"section,omitempty"`
	Priority    string   `yaml:"priority,omitempty"`
	Vendor      string   `yaml:"vendor,omitempty"`
	Homepage    string   `yaml:"homepage,omitempty"`
	Maintainer  string   `yaml:"maintainer,omitempty"`
	Description string   `yaml:"description,omitempty"`
	License     string   `yaml:"license,omitempty"`
	Bindir      string   `yaml:"bindir,omitempty"`
	Meta        bool     `yaml:"meta,omitempty"` // make package without binaries - only deps
}

// NFPMScripts is used to specify maintainer scripts.
type NFPMScripts struct {
	PreInstall  string `yaml:"preinstall,omitempty"`
	PostInstall string `yaml:"postinstall,omitempty"`
	PreRemove   string `yaml:"preremove,omitempty"`
	PostRemove  string `yaml:"postremove,omitempty"`
}

type NFPMRPMSignature struct {
	// PGP secret key, can be ASCII-armored
	KeyFile       string `yaml:"key_file,omitempty"`
	KeyPassphrase string `yaml:"-"` // populated from environment variable
}

// NFPMRPMScripts represents scripts only available on RPM packages.
type NFPMRPMScripts struct {
	PreTrans  string `yaml:"pretrans,omitempty"`
	PostTrans string `yaml:"posttrans,omitempty"`
}

// NFPMRPM is custom configs that are only available on RPM packages.
type NFPMRPM struct {
	Summary     string           `yaml:"summary,omitempty"`
	Group       string           `yaml:"group,omitempty"`
	Compression string           `yaml:"compression,omitempty"`
	Signature   NFPMRPMSignature `yaml:"signature,omitempty"`
	Scripts     NFPMRPMScripts   `yaml:"scripts,omitempty"`
}

// NFPMDebScripts is scripts only available on deb packages.
type NFPMDebScripts struct {
	Rules     string `yaml:"rules,omitempty"`
	Templates string `yaml:"templates,omitempty"`
}

// NFPMDebTriggers contains triggers only available for deb packages.
// https://wiki.debian.org/DpkgTriggers
// https://man7.org/linux/man-pages/man5/deb-triggers.5.html
type NFPMDebTriggers struct {
	Interest        []string `yaml:"interest,omitempty"`
	InterestAwait   []string `yaml:"interest_await,omitempty"`
	InterestNoAwait []string `yaml:"interest_noawait,omitempty"`
	Activate        []string `yaml:"activate,omitempty"`
	ActivateAwait   []string `yaml:"activate_await,omitempty"`
	ActivateNoAwait []string `yaml:"activate_noawait,omitempty"`
}

// NFPMDebSignature contains config for signing deb packages created by nfpm.
type NFPMDebSignature struct {
	// PGP secret key, can be ASCII-armored
	KeyFile       string `yaml:"key_file,omitempty"`
	KeyPassphrase string `yaml:"-"` // populated from environment variable
	// origin, maint or archive (defaults to origin)
	Type string `yaml:"type,omitempty"`
}

// NFPMDeb is custom configs that are only available on deb packages.
type NFPMDeb struct {
	Scripts   NFPMDebScripts   `yaml:"scripts,omitempty"`
	Triggers  NFPMDebTriggers  `yaml:"triggers,omitempty"`
	Breaks    []string         `yaml:"breaks,omitempty"`
	Signature NFPMDebSignature `yaml:"signature,omitempty"`
	Lintian   []string         `yaml:"lintian_overrides,omitempty"`
}

type NFPMAPKScripts struct {
	PreUpgrade  string `yaml:"preupgrade,omitempty"`
	PostUpgrade string `yaml:"postupgrade,omitempty"`
}

// NFPMAPKSignature contains config for signing apk packages created by nfpm.
type NFPMAPKSignature struct {
	// RSA private key in PEM format
	KeyFile       string `yaml:"key_file,omitempty"`
	KeyPassphrase string `yaml:"-"` // populated from environment variable
	// defaults to <maintainer email>.rsa.pub
	KeyName string `yaml:"key_name,omitempty"`
}

// NFPMAPK is custom config only available on apk packages.
type NFPMAPK struct {
	Scripts   NFPMAPKScripts   `yaml:"scripts,omitempty"`
	Signature NFPMAPKSignature `yaml:"signature,omitempty"`
}

// NFPMOverridables is used to specify per package format settings.
type NFPMOverridables struct {
	FileNameTemplate string            `yaml:"file_name_template,omitempty"`
	PackageName      string            `yaml:"package_name,omitempty"`
	Epoch            string            `yaml:"epoch,omitempty"`
	Release          string            `yaml:"release,omitempty"`
	Prerelease       string            `yaml:"prerelease,omitempty"`
	VersionMetadata  string            `yaml:"version_metadata,omitempty"`
	Replacements     map[string]string `yaml:"replacements,omitempty"`
	Dependencies     []string          `yaml:"dependencies,omitempty"`
	Recommends       []string          `yaml:"recommends,omitempty"`
	Suggests         []string          `yaml:"suggests,omitempty"`
	Conflicts        []string          `yaml:"conflicts,omitempty"`
	Replaces         []string          `yaml:"replaces,omitempty"`
	EmptyFolders     []string          `yaml:"empty_folders,omitempty"` // deprecated
	Contents         files.Contents    `yaml:"contents,omitempty"`
	Scripts          NFPMScripts       `yaml:"scripts,omitempty"`
	RPM              NFPMRPM           `yaml:"rpm,omitempty"`
	Deb              NFPMDeb           `yaml:"deb,omitempty"`
	APK              NFPMAPK           `yaml:"apk,omitempty"`
}

// SBOM config.
type SBOM struct {
	ID        string   `yaml:"id,omitempty"`
	Cmd       string   `yaml:"cmd,omitempty"`
	Env       []string `yaml:"env,omitempty"`
	Args      []string `yaml:"args,omitempty"`
	Documents []string `yaml:"documents,omitempty"`
	Artifacts string   `yaml:"artifacts,omitempty"`
	IDs       []string `yaml:"ids,omitempty"`
}

// Sign config.
type Sign struct {
	ID          string   `yaml:"id,omitempty"`
	Cmd         string   `yaml:"cmd,omitempty"`
	Args        []string `yaml:"args,omitempty"`
	Signature   string   `yaml:"signature,omitempty"`
	Artifacts   string   `yaml:"artifacts,omitempty"`
	IDs         []string `yaml:"ids,omitempty"`
	Stdin       *string  `yaml:"stdin,omitempty"`
	StdinFile   string   `yaml:"stdin_file,omitempty"`
	Env         []string `yaml:"env,omitempty"`
	Certificate string   `yaml:"certificate,omitempty"`
	Output      bool     `yaml:"output,omitempty"`
}

// SnapcraftAppMetadata for the binaries that will be in the snap package.
type SnapcraftAppMetadata struct {
	Plugs            []string
	Daemon           string
	Args             string
	Completer        string `yaml:"completer,omitempty"`
	Command          string `yaml:"command"`
	RestartCondition string `yaml:"restart_condition,omitempty"`
}

type SnapcraftLayoutMetadata struct {
	Symlink  string `yaml:"symlink,omitempty"`
	Bind     string `yaml:"bind,omitempty"`
	BindFile string `yaml:"bind_file,omitempty"`
	Type     string `yaml:"type,omitempty"`
}

// Snapcraft config.
type Snapcraft struct {
	NameTemplate string            `yaml:"name_template,omitempty"`
	Replacements map[string]string `yaml:"replacements,omitempty"`
	Publish      bool              `yaml:"publish,omitempty"`

	ID               string                             `yaml:"id,omitempty"`
	Builds           []string                           `yaml:"builds,omitempty"`
	Name             string                             `yaml:"name,omitempty"`
	Summary          string                             `yaml:"summary,omitempty"`
	Description      string                             `yaml:"description,omitempty"`
	Base             string                             `yaml:"base,omitempty"`
	License          string                             `yaml:"license,omitempty"`
	Grade            string                             `yaml:"grade,omitempty"`
	ChannelTemplates []string                           `yaml:"channel_templates,omitempty"`
	Confinement      string                             `yaml:"confinement,omitempty"`
	Layout           map[string]SnapcraftLayoutMetadata `yaml:"layout,omitempty"`
	Apps             map[string]SnapcraftAppMetadata    `yaml:"apps,omitempty"`
	Plugs            map[string]interface{}             `yaml:"plugs,omitempty"`

	Files []SnapcraftExtraFiles `yaml:"extra_files,omitempty"`
}

// SnapcraftExtraFiles config.
type SnapcraftExtraFiles struct {
	Source      string `yaml:"source"`
	Destination string `yaml:"destination,omitempty"`
	Mode        uint32 `yaml:"mode,omitempty"`
}

// Snapshot config.
type Snapshot struct {
	NameTemplate string `yaml:"name_template,omitempty"`
}

// Checksum config.
type Checksum struct {
	NameTemplate string      `yaml:"name_template,omitempty"`
	Algorithm    string      `yaml:"algorithm,omitempty"`
	IDs          []string    `yaml:"ids,omitempty"`
	Disable      bool        `yaml:"disable,omitempty"`
	ExtraFiles   []ExtraFile `yaml:"extra_files,omitempty"`
}

// Docker image config.
type Docker struct {
	ID                 string   `yaml:"id,omitempty"`
	IDs                []string `yaml:"ids,omitempty"`
	Goos               string   `yaml:"goos,omitempty"`
	Goarch             string   `yaml:"goarch,omitempty"`
	Goarm              string   `yaml:"goarm,omitempty"`
	Dockerfile         string   `yaml:"dockerfile,omitempty"`
	ImageTemplates     []string `yaml:"image_templates,omitempty"`
	SkipPush           string   `yaml:"skip_push,omitempty"`
	Files              []string `yaml:"extra_files,omitempty"`
	BuildFlagTemplates []string `yaml:"build_flag_templates,omitempty"`
	PushFlags          []string `yaml:"push_flags,omitempty"`
	Buildx             bool     `yaml:"use_buildx,omitempty"` // deprecated: use Use instead
	Use                string   `yaml:"use,omitempty"`
}

// DockerManifest config.
type DockerManifest struct {
	ID             string   `yaml:"id,omitempty"`
	NameTemplate   string   `yaml:"name_template,omitempty"`
	SkipPush       string   `yaml:"skip_push,omitempty"`
	ImageTemplates []string `yaml:"image_templates,omitempty"`
	CreateFlags    []string `yaml:"create_flags,omitempty"`
	PushFlags      []string `yaml:"push_flags,omitempty"`
	Use            string   `yaml:"use,omitempty"`
}

// Filters config.
type Filters struct {
	Exclude []string `yaml:"exclude,omitempty"`
}

// Changelog Config.
type Changelog struct {
	Filters Filters          `yaml:"filters,omitempty"`
	Sort    string           `yaml:"sort,omitempty"`
	Skip    bool             `yaml:"skip,omitempty"` // TODO(caarlos0): rename to Disable to match other pipes
	Use     string           `yaml:"use,omitempty" jsonschema:"enum=git,enum=github,enum=github-native,enum=gitlab,default=git"`
	Groups  []ChangeLogGroup `yaml:"groups,omitempty"`
}

// ChangeLogGroup holds the grouping criteria for the changelog.
type ChangeLogGroup struct {
	Title  string `yaml:"title,omitempty"`
	Regexp string `yaml:"regexp,omitempty"`
	Order  int    `yaml:"order,omitempty"`
}

// EnvFiles holds paths to files that contains environment variables
// values like the github token for example.
type EnvFiles struct {
	GitHubToken string `yaml:"github_token,omitempty"`
	GitLabToken string `yaml:"gitlab_token,omitempty"`
	GiteaToken  string `yaml:"gitea_token,omitempty"`
}

// Before config.
type Before struct {
	Hooks []string `yaml:"hooks,omitempty"`
}

// Blob contains config for GO CDK blob.
type Blob struct {
	Bucket     string      `yaml:"bucket,omitempty"`
	Provider   string      `yaml:"provider,omitempty"`
	Region     string      `yaml:"region,omitempty"`
	DisableSSL bool        `yaml:"disableSSL,omitempty"` // nolint:tagliatelle // TODO(caarlos0): rename to disable_ssl
	Folder     string      `yaml:"folder,omitempty"`
	KMSKey     string      `yaml:"kmskey,omitempty"`
	IDs        []string    `yaml:"ids,omitempty"`
	Endpoint   string      `yaml:"endpoint,omitempty"` // used for minio for example
	ExtraFiles []ExtraFile `yaml:"extra_files,omitempty"`
}

// Upload configuration.
type Upload struct {
	Name               string            `yaml:"name,omitempty"`
	IDs                []string          `yaml:"ids,omitempty"`
	Target             string            `yaml:"target,omitempty"`
	Username           string            `yaml:"username,omitempty"`
	Mode               string            `yaml:"mode,omitempty"`
	Method             string            `yaml:"method,omitempty"`
	ChecksumHeader     string            `yaml:"checksum_header,omitempty"`
	TrustedCerts       string            `yaml:"trusted_certificates,omitempty"`
	Checksum           bool              `yaml:"checksum,omitempty"`
	Signature          bool              `yaml:"signature,omitempty"`
	CustomArtifactName bool              `yaml:"custom_artifact_name,omitempty"`
	CustomHeaders      map[string]string `yaml:"custom_headers,omitempty"`
}

// Publisher configuration.
type Publisher struct {
	Name       string      `yaml:"name,omitempty"`
	IDs        []string    `yaml:"ids,omitempty"`
	Checksum   bool        `yaml:"checksum,omitempty"`
	Signature  bool        `yaml:"signature,omitempty"`
	Dir        string      `yaml:"dir,omitempty"`
	Cmd        string      `yaml:"cmd,omitempty"`
	Env        []string    `yaml:"env,omitempty"`
	ExtraFiles []ExtraFile `yaml:"extra_files,omitempty"`
}

// Source configuration.
type Source struct {
	NameTemplate   string `yaml:"name_template,omitempty"`
	Format         string `yaml:"format,omitempty"`
	Enabled        bool   `yaml:"enabled,omitempty"`
	PrefixTemplate string `yaml:"prefix_template,omitempty"`
}

// Project includes all project configuration.
type Project struct {
	ProjectName     string           `yaml:"project_name,omitempty"`
	Env             []string         `yaml:"env,omitempty"`
	Release         Release          `yaml:"release,omitempty"`
	Milestones      []Milestone      `yaml:"milestones,omitempty"`
	Brews           []Homebrew       `yaml:"brews,omitempty"`
	Rigs            []GoFish         `yaml:"rigs,omitempty"`
	AURs            []AUR            `yaml:"aurs,omitempty"`
	Krews           []Krew           `yaml:"krews,omitempty"`
	Scoop           Scoop            `yaml:"scoop,omitempty"`
	Builds          []Build          `yaml:"builds,omitempty"`
	Archives        []Archive        `yaml:"archives,omitempty"`
	NFPMs           []NFPM           `yaml:"nfpms,omitempty"`
	Snapcrafts      []Snapcraft      `yaml:"snapcrafts,omitempty"`
	Snapshot        Snapshot         `yaml:"snapshot,omitempty"`
	Checksum        Checksum         `yaml:"checksum,omitempty"`
	Dockers         []Docker         `yaml:"dockers,omitempty"`
	DockerManifests []DockerManifest `yaml:"docker_manifests,omitempty"`
	Artifactories   []Upload         `yaml:"artifactories,omitempty"`
	Uploads         []Upload         `yaml:"uploads,omitempty"`
	Blobs           []Blob           `yaml:"blobs,omitempty"`
	Publishers      []Publisher      `yaml:"publishers,omitempty"`
	Changelog       Changelog        `yaml:"changelog,omitempty"`
	Dist            string           `yaml:"dist,omitempty"`
	Signs           []Sign           `yaml:"signs,omitempty"`
	DockerSigns     []Sign           `yaml:"docker_signs,omitempty"`
	EnvFiles        EnvFiles         `yaml:"env_files,omitempty"`
	Before          Before           `yaml:"before,omitempty"`
	Source          Source           `yaml:"source,omitempty"`
	GoMod           GoMod            `yaml:"gomod,omitempty"`
	Announce        Announce         `yaml:"announce,omitempty"`
	SBOMs           []SBOM           `yaml:"sboms,omitempty"`

	UniversalBinaries []UniversalBinary `yaml:"universal_binaries,omitempty"`

	// this is a hack ¯\_(ツ)_/¯
	SingleBuild Build `yaml:"build,omitempty"`

	// should be set if using github enterprise
	GitHubURLs GitHubURLs `yaml:"github_urls,omitempty"`

	// should be set if using a private gitlab
	GitLabURLs GitLabURLs `yaml:"gitlab_urls,omitempty"`

	// should be set if using Gitea
	GiteaURLs GiteaURLs `yaml:"gitea_urls,omitempty"`
}

type GoMod struct {
	Proxy    bool     `yaml:"proxy,omitempty"`
	Env      []string `yaml:"env,omitempty"`
	GoBinary string   `yaml:"gobinary,omitempty"`
}

type Announce struct {
	Skip       string     `yaml:"skip,omitempty"`
	Twitter    Twitter    `yaml:"twitter,omitempty"`
	Reddit     Reddit     `yaml:"reddit,omitempty"`
	Slack      Slack      `yaml:"slack,omitempty"`
	Discord    Discord    `yaml:"discord,omitempty"`
	Teams      Teams      `yaml:"teams,omitempty"`
	SMTP       SMTP       `yaml:"smtp,omitempty"`
	Mattermost Mattermost `yaml:"mattermost,omitempty"`
	LinkedIn   LinkedIn   `yaml:"linkedin,omitempty"`
	Telegram   Telegram   `yaml:"telegram,omitempty"`
	Webhook    Webhook    `yaml:"webhook,omitempty"`
}

type Webhook struct {
	Enabled         bool              `yaml:"enabled,omitempty"`
	SkipTLSVerify   bool              `yaml:"skip_tls_verify,omitempty"`
	MessageTemplate string            `yaml:"message_template,omitempty"`
	EndpointURL     string            `yaml:"endpoint_url,omitempty"`
	Headers         map[string]string `yaml:"headers,omitempty"`
	ContentType     string            `yaml:"content_type,omitempty"`
}

type Twitter struct {
	Enabled         bool   `yaml:"enabled,omitempty"`
	MessageTemplate string `yaml:"message_template,omitempty"`
}

type Reddit struct {
	Enabled       bool   `yaml:"enabled,omitempty"`
	ApplicationID string `yaml:"application_id,omitempty"`
	Username      string `yaml:"username,omitempty"`
	TitleTemplate string `yaml:"title_template,omitempty"`
	URLTemplate   string `yaml:"url_template,omitempty"`
	Sub           string `yaml:"sub,omitempty"`
}

type Slack struct {
	Enabled         bool   `yaml:"enabled,omitempty"`
	MessageTemplate string `yaml:"message_template,omitempty"`
	Channel         string `yaml:"channel,omitempty"`
	Username        string `yaml:"username,omitempty"`
	IconEmoji       string `yaml:"icon_emoji,omitempty"`
	IconURL         string `yaml:"icon_url,omitempty"`
}

type Discord struct {
	Enabled         bool   `yaml:"enabled,omitempty"`
	MessageTemplate string `yaml:"message_template,omitempty"`
	Author          string `yaml:"author,omitempty"`
	Color           string `yaml:"color,omitempty"`
	IconURL         string `yaml:"icon_url,omitempty"`
}

type Teams struct {
	Enabled         bool   `yaml:"enabled,omitempty"`
	TitleTemplate   string `yaml:"title_template,omitempty"`
	MessageTemplate string `yaml:"message_template,omitempty"`
	Color           string `yaml:"color,omitempty"`
	IconURL         string `yaml:"icon_url,omitempty"`
}

type Mattermost struct {
	Enabled         bool   `yaml:"enabled,omitempty"`
	MessageTemplate string `yaml:"message_template,omitempty"`
	TitleTemplate   string `yaml:"title_template,omitempty"`
	Color           string `yaml:"color,omitempty"`
	Channel         string `yaml:"channel,omitempty"`
	Username        string `yaml:"username,omitempty"`
	IconEmoji       string `yaml:"icon_emoji,omitempty"`
	IconURL         string `yaml:"icon_url,omitempty"`
}

type SMTP struct {
	Enabled            bool     `yaml:"enabled,omitempty"`
	Host               string   `yaml:"host,omitempty"`
	Port               int      `yaml:"port,omitempty"`
	Username           string   `yaml:"username,omitempty"`
	From               string   `yaml:"from,omitempty"`
	To                 []string `yaml:"to,omitempty"`
	SubjectTemplate    string   `yaml:"subject_template,omitempty"`
	BodyTemplate       string   `yaml:"body_template,omitempty"`
	InsecureSkipVerify bool     `yaml:"insecure_skip_verify,omitempty"`
}

type LinkedIn struct {
	Enabled         bool   `yaml:"enabled,omitempty"`
	MessageTemplate string `yaml:"message_template,omitempty"`
}

type Telegram struct {
	Enabled         bool   `yaml:"enabled,omitempty"`
	MessageTemplate string `yaml:"message_template,omitempty"`
	ChatID          int64  `yaml:"chat_id,omitempty"`
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
