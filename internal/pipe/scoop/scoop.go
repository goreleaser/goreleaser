// Package scoop provides a Pipe that generates a scoop.sh App Manifest and pushes it to a repo
package scoop

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/pipe/release"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
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
	client, err := client.New(ctx)
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
	if ctx.Config.Scoop.Repo.Name == "" {
		return pipe.Skip("scoop section is not configured")
	}
	if ctx.Config.Archive.Format == "binary" {
		return pipe.Skip("archive format is binary")
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

	content, err := buildManifest(ctx, archives)
	if err != nil {
		return err
	}

	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}
	if ctx.Config.Release.Draft {
		return pipe.Skip("release is marked as draft")
	}
	return client.CreateFile(
		ctx,
		ctx.Config.Scoop.CommitAuthor,
		ctx.Config.Scoop.Repo,
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
	Persist      []string            `json:"persist,omitempty"`     // Persist data between updates
}

// Resource represents a combination of a url and a binary name for an architecture
type Resource struct {
	URL  string `json:"url"`  // URL to the archive
	Bin  string `json:"bin"`  // name of binary inside the archive
	Hash string `json:"hash"` // the archive checksum
}

func buildManifest(ctx *context.Context, artifacts []artifact.Artifact) (bytes.Buffer, error) {
	var result bytes.Buffer
	var manifest = Manifest{
		Version:      ctx.Version,
		Architecture: make(map[string]Resource),
		Homepage:     ctx.Config.Scoop.Homepage,
		License:      ctx.Config.Scoop.License,
		Description:  ctx.Config.Scoop.Description,
		Persist:      ctx.Config.Scoop.Persist,
	}

	for _, artifact := range artifacts {
		var arch = "64bit"
		if artifact.Goarch == "386" {
			arch = "32bit"
		}
		urlTemplate := ctx.Config.Scoop.URLTemplate
		if ctx.Config.Scoop.URLTemplate == "" {
			artifactPath, ok := artifact.Extra[release.ArtifactDownloadPath]
			if !ok {
				return result, fmt.Errorf("artifact (%s) missing ArtifactDownloadPath", artifact.Name)
			}
			urlTemplate = fmt.Sprintf("%s%s",
				ctx.Config.RepoURLs.Download,
				artifactPath)
		}

		url, err := tmpl.New(ctx).
			WithArtifact(artifact, map[string]string{}).
			Apply(urlTemplate)
		if err != nil {
			return result, err
		}

		sum, err := artifact.Checksum()
		if err != nil {
			return result, err
		}

		manifest.Architecture[arch] = Resource{
			URL:  url,
			Bin:  ctx.Config.Builds[0].Binary + ".exe",
			Hash: sum,
		}
	}

	data, err := json.MarshalIndent(manifest, "", "    ")
	if err != nil {
		return result, err
	}
	_, err = result.Write(data)
	return result, err
}
