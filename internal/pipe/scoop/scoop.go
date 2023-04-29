// Package scoop provides a Pipe that generates a scoop.sh App Manifest and pushes it to a bucket
package scoop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/commitauthor"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// ErrNoWindows when there is no build for windows (goos doesn't contain
// windows) or archive.format is binary.
type ErrNoWindows struct {
	goamd64 string
}

func (e ErrNoWindows) Error() string {
	return fmt.Sprintf("scoop requires a windows archive, but no archives matched goos=windows goarch=[386 amd64] goamd64=%s\nLearn more at https://goreleaser.com/errors/scoop-archive\n", e.goamd64) // nolint: revive
}

const scoopConfigExtra = "ScoopConfig"

// Pipe that builds and publishes scoop manifests.
type Pipe struct{}

func (Pipe) String() string                 { return "scoop manifests" }
func (Pipe) Skip(ctx *context.Context) bool { return ctx.Config.Scoop.Bucket.Name == "" }

// Run creates the scoop manifest locally.
func (Pipe) Run(ctx *context.Context) error {
	client, err := client.New(ctx)
	if err != nil {
		return err
	}
	return doRun(ctx, client)
}

// Publish scoop manifest.
func (Pipe) Publish(ctx *context.Context) error {
	client, err := client.New(ctx)
	if err != nil {
		return err
	}
	return doPublish(ctx, client)
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Scoop.Name == "" {
		ctx.Config.Scoop.Name = ctx.Config.ProjectName
	}
	ctx.Config.Scoop.CommitAuthor = commitauthor.Default(ctx.Config.Scoop.CommitAuthor)
	if ctx.Config.Scoop.CommitMessageTemplate == "" {
		ctx.Config.Scoop.CommitMessageTemplate = "Scoop update for {{ .ProjectName }} version {{ .Tag }}"
	}
	if ctx.Config.Scoop.Goamd64 == "" {
		ctx.Config.Scoop.Goamd64 = "v1"
	}
	return nil
}

func doRun(ctx *context.Context, cl client.Client) error {
	scoop := ctx.Config.Scoop

	archives := ctx.Artifacts.Filter(
		artifact.And(
			artifact.ByGoos("windows"),
			artifact.ByType(artifact.UploadableArchive),
			artifact.Or(
				artifact.And(
					artifact.ByGoarch("amd64"),
					artifact.ByGoamd64(scoop.Goamd64),
				),
				artifact.ByGoarch("386"),
			),
		),
	).List()
	if len(archives) == 0 {
		return ErrNoWindows{scoop.Goamd64}
	}

	filename := scoop.Name + ".json"

	data, err := dataFor(ctx, cl, archives)
	if err != nil {
		return err
	}
	content, err := doBuildManifest(data)
	if err != nil {
		return err
	}

	path := filepath.Join(ctx.Config.Dist, filename)
	log.WithField("manifest", path).Info("writing")
	if err := os.WriteFile(path, content.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write scoop manifest: %w", err)
	}

	ctx.Artifacts.Add(&artifact.Artifact{
		Name: filename,
		Path: path,
		Type: artifact.ScoopManifest,
		Extra: map[string]interface{}{
			scoopConfigExtra: scoop,
		},
	})
	return nil
}

func doPublish(ctx *context.Context, cl client.Client) error {
	manifests := ctx.Artifacts.Filter(artifact.ByType(artifact.ScoopManifest)).List()
	if len(manifests) == 0 { // should never happen
		return nil
	}

	manifest := manifests[0]

	scoop, err := artifact.Extra[config.Scoop](*manifest, scoopConfigExtra)
	if err != nil {
		return err
	}

	if strings.TrimSpace(scoop.SkipUpload) == "true" {
		return pipe.Skip("scoop.skip_upload is true")
	}
	if strings.TrimSpace(scoop.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("release is prerelease")
	}

	relDisabled, err := tmpl.New(ctx).Bool(ctx.Config.Release.Disable)
	if err != nil {
		return err
	}
	if relDisabled {
		return pipe.Skip("release is disabled")
	}

	commitMessage, err := tmpl.New(ctx).Apply(scoop.CommitMessageTemplate)
	if err != nil {
		return err
	}

	author, err := commitauthor.Get(ctx, scoop.CommitAuthor)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(manifest.Path)
	if err != nil {
		return err
	}

	ref, err := client.TemplateRef(tmpl.New(ctx).Apply, scoop.Bucket)
	if err != nil {
		return err
	}
	scoop.Bucket = ref

	repo := client.RepoFromRef(scoop.Bucket)
	gpath := path.Join(scoop.Folder, manifest.Name)

	if scoop.Bucket.Git.URL != "" {
		return client.NewGitUploadClient(ctx, repo.Branch).
			CreateFile(ctx, author, repo, content, gpath, commitMessage)
	}

	cl, err = client.NewIfToken(ctx, cl, scoop.Bucket.Token)
	if err != nil {
		return err
	}

	if !scoop.Bucket.PullRequest.Enabled {
		return cl.CreateFile(ctx, author, repo, content, gpath, commitMessage)
	}

	log.Info("brews.pull_request enabled, creating a PR")
	pcl, ok := cl.(client.PullRequestOpener)
	if !ok {
		return fmt.Errorf("client does not support pull requests")
	}

	if err := cl.CreateFile(ctx, author, repo, content, gpath, commitMessage); err != nil {
		return err
	}

	title := fmt.Sprintf("Updated %s to %s", ctx.Config.ProjectName, ctx.Version)
	return pcl.OpenPullRequest(ctx, repo, scoop.Bucket.PullRequest.Base, title)
}

// Manifest represents a scoop.sh App Manifest.
// more info: https://github.com/lukesampson/scoop/wiki/App-Manifests
type Manifest struct {
	Version      string              `json:"version"`                // The version of the app that this manifest installs.
	Architecture map[string]Resource `json:"architecture"`           // `architecture`: If the app has 32- and 64-bit versions, architecture can be used to wrap the differences.
	Homepage     string              `json:"homepage,omitempty"`     // `homepage`: The home page for the program.
	License      string              `json:"license,omitempty"`      // `license`: The software license for the program. For well-known licenses, this will be a string like "MIT" or "GPL2". For custom licenses, this should be the URL of the license.
	Description  string              `json:"description,omitempty"`  // Description of the app
	Persist      []string            `json:"persist,omitempty"`      // Persist data between updates
	PreInstall   []string            `json:"pre_install,omitempty"`  // An array of strings, of the commands to be executed before an application is installed.
	PostInstall  []string            `json:"post_install,omitempty"` // An array of strings, of the commands to be executed after an application is installed.
	Depends      []string            `json:"depends,omitempty"`      // A string or an array of strings.
	Shortcuts    [][]string          `json:"shortcuts,omitempty"`    // A two-dimensional array of string, specifies the shortcut values to make available in the startmenu.
}

// Resource represents a combination of a url and a binary name for an architecture.
type Resource struct {
	URL  string   `json:"url"`  // URL to the archive
	Bin  []string `json:"bin"`  // name of binary inside the archive
	Hash string   `json:"hash"` // the archive checksum
}

func doBuildManifest(manifest Manifest) (bytes.Buffer, error) {
	var result bytes.Buffer
	data, err := json.MarshalIndent(manifest, "", "    ")
	if err != nil {
		return result, err
	}
	_, err = result.Write(data)
	return result, err
}

func dataFor(ctx *context.Context, cl client.Client, artifacts []*artifact.Artifact) (Manifest, error) {
	manifest := Manifest{
		Version:      ctx.Version,
		Architecture: map[string]Resource{},
		Homepage:     ctx.Config.Scoop.Homepage,
		License:      ctx.Config.Scoop.License,
		Description:  ctx.Config.Scoop.Description,
		Persist:      ctx.Config.Scoop.Persist,
		PreInstall:   ctx.Config.Scoop.PreInstall,
		PostInstall:  ctx.Config.Scoop.PostInstall,
		Depends:      ctx.Config.Scoop.Depends,
		Shortcuts:    ctx.Config.Scoop.Shortcuts,
	}

	if ctx.Config.Scoop.URLTemplate == "" {
		url, err := cl.ReleaseURLTemplate(ctx)
		if err != nil {
			return manifest, err
		}
		ctx.Config.Scoop.URLTemplate = url
	}

	for _, artifact := range artifacts {
		if artifact.Goos != "windows" {
			continue
		}

		var arch string
		switch {
		case artifact.Goarch == "386":
			arch = "32bit"
		case artifact.Goarch == "amd64":
			arch = "64bit"
		default:
			continue
		}

		url, err := tmpl.New(ctx).WithArtifact(artifact).Apply(ctx.Config.Scoop.URLTemplate)
		if err != nil {
			return manifest, err
		}

		sum, err := artifact.Checksum("sha256")
		if err != nil {
			return manifest, err
		}

		log.WithFields(log.Fields{
			"artifactExtras":   artifact.Extra,
			"fromURLTemplate":  ctx.Config.Scoop.URLTemplate,
			"templatedBrewURL": url,
			"sum":              sum,
		}).Debug("scoop url templating")

		binaries, err := binaries(*artifact)
		if err != nil {
			return manifest, err
		}

		manifest.Architecture[arch] = Resource{
			URL:  url,
			Bin:  binaries,
			Hash: sum,
		}
	}

	return manifest, nil
}

func binaries(a artifact.Artifact) ([]string, error) {
	// nolint: prealloc
	var bins []string
	wrap := artifact.ExtraOr(a, artifact.ExtraWrappedIn, "")
	builds, err := artifact.Extra[[]artifact.Artifact](a, artifact.ExtraBuilds)
	if err != nil {
		return nil, err
	}
	for _, b := range builds {
		bins = append(bins, filepath.Join(wrap, b.Name))
	}
	return bins, nil
}
