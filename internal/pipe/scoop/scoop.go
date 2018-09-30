// Package scoop provides a Pipe that generates a scoop.sh App Manifest and pushes it to a bucket
package scoop

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/deprecate"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// ErrNoWindows when there is no build for windows (goos doesn't contain windows)
// or the windows builds were not archived as tar.gz/zip (build mode is binary)
var ErrNoWindows = errors.New("scoop requires a windows build and a zip or tar.gz archive")

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
	if !reflect.DeepEqual(ctx.Config.OldScoop, config.Scoop{}) {
		deprecate.Notice("scoop")
		ctx.Config.Scoops = append(ctx.Config.Scoops, ctx.Config.OldScoop)
	}
	for i, scoop := range ctx.Config.Scoops {
		if scoop.Name == "" {
			scoop.Name = ctx.Config.ProjectName
		}
		if scoop.CommitAuthor.Name == "" {
			scoop.CommitAuthor.Name = "goreleaserbot"
		}
		if scoop.CommitAuthor.Email == "" {
			scoop.CommitAuthor.Email = "goreleaser@carlosbecker.com"
		}
		if scoop.URLTemplate == "" {
			scoop.URLTemplate = fmt.Sprintf(
				"%s/%s/%s/releases/download/{{ .Tag }}/{{ .ArtifactName }}",
				ctx.Config.GitHubURLs.Download,
				ctx.Config.Release.GitHub.Owner,
				ctx.Config.Release.GitHub.Name,
			)
		}
		ctx.Config.Scoops[i] = scoop
	}
	return nil
}

func doRun(ctx *context.Context, client client.Client) error {
	if len(ctx.Config.Scoops) == 0 {
		return pipe.Skip("scoop section is not configured")
	}

	var g = semerrgroup.New(ctx.Parallelism)
	for _, scoop := range ctx.Config.Scoops {
		scoop := scoop
		g.Go(func() error {
			return doRunScoop(ctx, client, scoop)
		})
	}
	return g.Wait()
}

func doRunScoop(ctx *context.Context, client client.Client, scoop config.Scoop) error {
	var filter = artifact.And(
		artifact.ByGoos("windows"),
		artifact.ByType(artifact.UploadableArchive),
	)

	if len(scoop.Binaries) > 0 {
		filter = artifact.And(filter, artifact.ByBinaryName(scoop.Binaries...))
	}

	var archives = ctx.Artifacts.Filter(filter).List()

	// TODO: fix this
	// if ctx.Config.Archive.Format == "binary" {
	// 	return pipe.Skip("archive format is binary")
	// }
	if len(archives) == 0 {
		return ErrNoWindows
	}

	var path = scoop.Name + ".json"
	content, err := buildManifest(ctx, archives, scoop)
	if err != nil {
		return err
	}

	// TODO: this should be the first thing checked!
	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}
	if ctx.Config.Release.Draft {
		return pipe.Skip("release is marked as draft")
	}
	return client.CreateFile(
		ctx,
		scoop.CommitAuthor,
		scoop.Bucket,
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

func buildManifest(ctx *context.Context, artifacts []artifact.Artifact, scoop config.Scoop) (bytes.Buffer, error) {
	var result bytes.Buffer
	var manifest = Manifest{
		Version:      ctx.Version,
		Architecture: make(map[string]Resource),
		Homepage:     scoop.Homepage,
		License:      scoop.License,
		Description:  scoop.Description,
		Persist:      scoop.Persist,
	}

	for _, artifact := range artifacts {
		var arch = "64bit"
		if artifact.Goarch == "386" {
			arch = "32bit"
		}

		url, err := tmpl.New(ctx).
			WithArtifact(artifact, map[string]string{}).
			Apply(scoop.URLTemplate)
		if err != nil {
			return result, err
		}

		sum, err := artifact.Checksum()
		if err != nil {
			return result, err
		}

		manifest.Architecture[arch] = Resource{
			URL:  url,
			Bin:  ctx.Config.Builds[0].Binary + ".exe", // TODO: this is wrong
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
