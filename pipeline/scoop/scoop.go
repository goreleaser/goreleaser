// Package scoop provides a Pipe that generates a scoop.sh App Manifest and pushes it to a bucket
package scoop

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/archiveformat"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/pipeline"
)

// ErrNoWindows64Build when there is no build for windows_amd64 (goos doesn't
// contain windows and/or goarch doesn't contain amd64)
var ErrNoWindows64Build = errors.New("scoop requires a windows amd64 build")

const platform = "windowsamd64"

// Pipe for build
type Pipe struct{}

// Description of the pipe
func (Pipe) String() string {
	return "Generating Scoop Manifest"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	client, err := client.NewGitHub(ctx)
	if err != nil {
		return err
	}
	return doRun(ctx, client)
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Scoop.CommitAuthor.Name == "" {
		ctx.Config.Scoop.CommitAuthor.Name = "goreleaserbot"
	}
	if ctx.Config.Scoop.CommitAuthor.Email == "" {
		ctx.Config.Scoop.CommitAuthor.Email = "goreleaser@carlosbecker.com"
	}
	return nil
}

func doRun(ctx *context.Context, client client.Client) error {
	if !ctx.Publish {
		return pipeline.Skip("--skip-publish is set")
	}
	if ctx.Config.Scoop.Bucket.Name == "" {
		return pipeline.Skip("scoop section is not configured")
	}
	if ctx.Config.Release.Draft {
		return pipeline.Skip("release is marked as draft")
	}
	if ctx.Config.Archive.Format == "binary" {
		return pipeline.Skip("archive format is binary")
	}

	var group = ctx.Binaries["windowsamd64"]
	if group == nil {
		return ErrNoWindows64Build
	}
	var fileName string
	for f := range group {
		fileName = f
		break
	}

	path := ctx.Config.ProjectName + ".json"

	content, err := buildManifest(ctx, client, fileName)
	if err != nil {
		return err
	}
	return client.CreateFile(
		ctx,
		ctx.Config.Scoop.CommitAuthor,
		ctx.Config.Scoop.Bucket,
		content,
		path)
}

// Manifest represents a scoop.sh App Manifest, more info:
// https://github.com/lukesampson/scoop/wiki/App-Manifests
type Manifest struct {
	Version      string   `json:"version"`                // The version of the app that this manifest installs.
	URL          []string `json:"url"`                    // The URL or URLs of files to download.
	Architecture string   `json:"architecture,omitempty"` // `architecture`: If the app has 32- and 64-bit versions, architecture can be used to wrap the differences.
	AutoUpdate   string   `json:"autoupdate,omitempty"`   // autoupdate: Definition of how the manifest can be updated automatically.
	Bin          []string `json:"bin,omitempty"`          // `bin`: A string or array of strings of programs (executables or scripts) to make available on the user's path.
	CheckVersion string   `json:"checkver,omitempty"`     // checkver: App maintainers and developers can use the bin/checkver tool to check for updated versions of apps.
	Depends      string   `json:"depends,omitempty"`      // `depends`: Runtime dependencies for the app which will be installed automatically.
	EnvAddToPath string   `json:"env_add_path,omitempty"` // `env_add_path`: Add this directory to the user's path (or system path if `--global` is used). The directory is relative to the install directory and must be inside the install directory.
	EnvSet       string   `json:"env_set,omitempty"`      // `env_set`: Sets one or more environment variables for the user (or system if `--global` is used).
	ExtractDir   string   `json:"extract_dir,omitempty"`  // `extract_dir`: If `url` points to a compressed file (.zip, .7z, .tar, .gz, .lzma, and .lzh are supported), Scoop will extract just the directory specified from it.
	Hash         string   `json:"hash,omitempty"`         // `hash`: A string or array of strings with a file hash for each URL in `url`. Hashes are SHA256 by default, but you can use SHA1 or MD5 by prefixing the hash string with 'sha1:' or 'md5:'.
	Homepage     string   `json:"homepage,omitempty"`     // `homepage`: The home page for the program.
	Installer    string   `json:"installer,omitempty"`    // `installer`|`uninstaller`: Instructions for running a non-MSI installer.
	License      string   `json:"license,omitempty"`      // `license`: The software license for the program. For well-known licenses, this will be a string like "MIT" or "GPL2". For custom licenses, this should be the URL of the license.
	Notes        string   `json:"notes,omitempty"`        // `notes`: A string with a message to be displayed after installing the app.
	PreInstall   string   `json:"pre_install,omitempty"`  // `pre_install` | `post_install` : A string or array of strings of the commands to be executed before or after an application is installed. (Available variables: `$dir`, `$persist_dir`, `$version` many more (_check the `lib/install`
	PsModule     string   `json:"psmodule,omitempty"`     // `psmodule`: Install as a PowerShell module in `~/scoop/modules`.
	Shortcuts    string   `json:"shortcuts,omitempty"`    // `shortcuts`: Specifies the shortcut values to make available in the startmenu. The array specifies an executable/Label pair.
	Suggest      string   `json:"suggest,omitempty"`      // `suggest`: Display a message suggesting optional apps that provide complementary features.
	Persist      string   `json:"persist,omitempty"`      // `persist` A string or array of strings of directories and files to persist inside the data directory for the app.
	Description  string   `json:"persist,omitempty"`
}

func buildManifest(ctx *context.Context, client client.Client, fileName string) (result bytes.Buffer, err error) {
	var file = fileName + "." + archiveformat.For(ctx, platform)

	var githubURL = "https://github.com"
	if ctx.Config.GitHubURLs.Download != "" {
		githubURL = ctx.Config.GitHubURLs.Download
	}

	var urls []string
	urls = append(urls, getDownloadURL(
		githubURL,
		ctx.Config.Release.GitHub.Owner,
		ctx.Config.Release.GitHub.Name,
		ctx.Version,
		file))

	binaries := make([]string, len(ctx.Binaries["windowsamd64"]))
	var i = 0
	for _, binaryGroup := range ctx.Binaries["windowsamd64"] {
		for _, binary := range binaryGroup {
			binaries[i] = binary.Name
			i++
		}
	}

	manifest := Manifest{
		Version:     ctx.Version,
		URL:         urls,
		Bin:         binaries,
		Homepage:    ctx.Config.Scoop.Homepage,
		License:     ctx.Config.Scoop.License,
		Description: ctx.Config.Scoop.Description,
	}

	data, err := json.MarshalIndent(manifest, "", "    ")
	if err != nil {
		return
	}
	_, err = result.Write(data)

	return
}

func getDownloadURL(githubURL, owner, name, version, file string) (url string) {
	return fmt.Sprintf("%s/%s/%s/releases/download/%s/%s",
		githubURL,
		owner,
		name,
		version,
		file)
}
