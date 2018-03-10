// Package scoop provides a Pipe that generates a scoop.sh App Manifest and pushes it to a bucket
package scoop

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/pipeline"
)

// ErrNoWindows when there is no build for windows (goos doesn't contain windows)
var ErrNoWindows = errors.New("scoop requires a windows build")

// Pipe for build
type Pipe struct{}

func (Pipe) String() string {
	return "creating Scoop Manifest"
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
	if ctx.Config.Scoop.Bucket.Name == "" {
		return pipeline.Skip("scoop section is not configured")
	}
	if ctx.Config.Archive.Format == "binary" {
		return pipeline.Skip("archive format is binary")
	}

	var archives = ctx.Artifacts.Filter(
		artifact.And(
			artifact.ByGoos("windows"),
			artifact.ByType(artifact.UploadableArchive),
		),
	).List()
	if len(archives) == 0 {
		return ErrNoWindows
	}

	path := ctx.Config.ProjectName + ".json"

	content, err := buildManifest(ctx, client, archives)
	if err != nil {
		return err
	}

	if ctx.SkipPublish {
		return pipeline.ErrSkipPublishEnabled
	}
	if ctx.Config.Release.Draft {
		return pipeline.Skip("release is marked as draft")
	}
	return client.CreateFile(
		ctx,
		ctx.Config.Scoop.CommitAuthor,
		ctx.Config.Scoop.Bucket,
		content,
		path,
		fmt.Sprintf("Scoop update for %s version %s", ctx.Config.ProjectName, ctx.Git.CurrentTag),
	)
}

// Manifest represents a scoop.sh App Manifest, more info:
// https://github.com/lukesampson/scoop/wiki/App-Manifests
type Manifest struct {
	Version      string              `json:"version"`               // The version of the app that this manifest installs.
	Architecture map[string]Resource `json:"architecture"`          // `architecture`: If the app has 32- and 64-bit versions, architecture can be used to wrap the differences.
	Homepage     string              `json:"homepage,omitempty"`    // `homepage`: The home page for the program.
	License      string              `json:"license,omitempty"`     // `license`: The software license for the program. For well-known licenses, this will be a string like "MIT" or "GPL2". For custom licenses, this should be the URL of the license.
	Description  string              `json:"description,omitempty"` // Description of the app
}

// Resource represents a combination of a url and a binary name for an architecture
type Resource struct {
	URL string `json:"url"` // URL to the archive
	Bin string `json:"bin"` // name of binary inside the archive
}

func buildManifest(ctx *context.Context, client client.Client, artifacts []artifact.Artifact) (result bytes.Buffer, err error) {
	manifest := Manifest{
		Version:      ctx.Version,
		Architecture: make(map[string]Resource),
		Homepage:     ctx.Config.Scoop.Homepage,
		License:      ctx.Config.Scoop.License,
		Description:  ctx.Config.Scoop.Description,
	}

	for _, artifact := range artifacts {
		var arch = "64bit"
		if artifact.Goarch == "386" {
			arch = "32bit"
		}
		manifest.Architecture[arch] = Resource{
			URL: getDownloadURL(ctx, ctx.Config.GitHubURLs.Download, artifact.Name),
			Bin: ctx.Config.Builds[0].Binary + ".exe",
		}
	}

	data, err := json.MarshalIndent(manifest, "", "    ")
	if err != nil {
		return
	}
	_, err = result.Write(data)
	return
}

func getDownloadURL(ctx *context.Context, githubURL, file string) string {
	return fmt.Sprintf(
		"%s/%s/%s/releases/download/%s/%s",
		githubURL,
		ctx.Config.Release.GitHub.Owner,
		ctx.Config.Release.GitHub.Name,
		ctx.Git.CurrentTag,
		file,
	)
}
