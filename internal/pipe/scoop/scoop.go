// Package scoop provides a Pipe that generates a scoop.sh App Manifest and pushes it to a bucket.
package scoop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/commitauthor"
	"github.com/goreleaser/goreleaser/internal/deprecate"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// ErrIncorrectArchiveCount happens when a given filter evaluates 0 or more
// than 1 archives.
type ErrIncorrectArchiveCount struct {
	goamd64  string
	ids      []string
	archives []*artifact.Artifact
}

func (e ErrIncorrectArchiveCount) Error() string {
	b := strings.Builder{}

	_, _ = b.WriteString("scoop requires a single windows archive, ")
	if len(e.archives) == 0 {
		_, _ = b.WriteString("but no archives ")
	} else {
		_, _ = b.WriteString(fmt.Sprintf("but found %d archives ", len(e.archives)))
	}

	_, _ = b.WriteString(fmt.Sprintf("matching the given filters: goos=windows goarch=[386 amd64 arm64] goamd64=%s ids=%s", e.goamd64, e.ids))

	if len(e.archives) > 0 {
		names := make([]string, 0, len(e.archives))
		for _, a := range e.archives {
			names = append(names, a.Name)
		}
		_, _ = b.WriteString(fmt.Sprintf(": %s", names))
	}

	_, _ = b.WriteString("\nLearn more at https://goreleaser.com/errors/scoop-archive\n")
	return b.String()
}

const scoopConfigExtra = "ScoopConfig"

// Pipe that builds and publishes scoop manifests.
type Pipe struct{}

func (Pipe) String() string        { return "scoop manifests" }
func (Pipe) ContinueOnError() bool { return true }
func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Scoop) || (ctx.Config.Scoop.Repository.Name == "" && len(ctx.Config.Scoops) == 0)
}

// Run creates the scoop manifest locally.
func (Pipe) Run(ctx *context.Context) error {
	cli, err := client.NewReleaseClient(ctx)
	if err != nil {
		return err
	}
	return runAll(ctx, cli)
}

// Publish scoop manifest.
func (Pipe) Publish(ctx *context.Context) error {
	client, err := client.New(ctx)
	if err != nil {
		return err
	}
	return publishAll(ctx, client)
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	if !reflect.DeepEqual(ctx.Config.Scoop.Bucket, config.RepoRef{}) ||
		!reflect.DeepEqual(ctx.Config.Scoop.Repository, config.RepoRef{}) {
		deprecate.Notice(ctx, "scoop")
		ctx.Config.Scoops = append(ctx.Config.Scoops, ctx.Config.Scoop)
	}

	for i := range ctx.Config.Scoops {
		scoop := &ctx.Config.Scoops[i]
		if scoop.Name == "" {
			scoop.Name = ctx.Config.ProjectName
		}
		if scoop.Folder != "" {
			deprecate.Notice(ctx, "scoops.folder")
			scoop.Directory = scoop.Folder
		}
		scoop.CommitAuthor = commitauthor.Default(scoop.CommitAuthor)
		if scoop.CommitMessageTemplate == "" {
			scoop.CommitMessageTemplate = "Scoop update for {{ .ProjectName }} version {{ .Tag }}"
		}
		if scoop.Goamd64 == "" {
			scoop.Goamd64 = "v1"
		}
		if !reflect.DeepEqual(scoop.Bucket, config.RepoRef{}) {
			scoop.Repository = scoop.Bucket
			deprecate.Notice(ctx, "scoops.bucket")
		}
	}
	return nil
}

func runAll(ctx *context.Context, cl client.ReleaseURLTemplater) error {
	for _, scoop := range ctx.Config.Scoops {
		err := doRun(ctx, scoop, cl)
		if err != nil {
			return err
		}
	}
	return nil
}

func doRun(ctx *context.Context, scoop config.Scoop, cl client.ReleaseURLTemplater) error {
	filters := []artifact.Filter{
		artifact.ByGoos("windows"),
		artifact.ByType(artifact.UploadableArchive),
		artifact.Or(
			artifact.And(
				artifact.ByGoarch("amd64"),
				artifact.ByGoamd64(scoop.Goamd64),
			),
			artifact.ByGoarch("arm64"),
			artifact.ByGoarch("386"),
		),
	}

	if len(scoop.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(scoop.IDs...))
	}

	filtered := ctx.Artifacts.Filter(artifact.And(filters...))
	archives := filtered.List()
	for _, platArchives := range filtered.GroupByPlatform() {
		// there might be multiple archives, but only of for each platform
		if len(platArchives) != 1 {
			return ErrIncorrectArchiveCount{scoop.Goamd64, scoop.IDs, archives}
		}
	}
	// handle no archives found whatsoever
	if len(archives) == 0 {
		return ErrIncorrectArchiveCount{scoop.Goamd64, scoop.IDs, archives}
	}

	tp := tmpl.New(ctx)

	if err := tp.ApplyAll(
		&scoop.Name,
		&scoop.Description,
		&scoop.Homepage,
		&scoop.SkipUpload,
	); err != nil {
		return err
	}

	ref, err := client.TemplateRef(tmpl.New(ctx).Apply, scoop.Repository)
	if err != nil {
		return err
	}
	scoop.Repository = ref

	data, err := dataFor(ctx, scoop, cl, archives)
	if err != nil {
		return err
	}
	content, err := doBuildManifest(data)
	if err != nil {
		return err
	}

	filename := scoop.Name + ".json"
	path := filepath.Join(ctx.Config.Dist, "scoop", scoop.Directory, filename)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
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

func publishAll(ctx *context.Context, cli client.Client) error {
	// even if one of them skips, we run them all, and then show return the
	// skips all at once. this is needed so we actually create the
	// `dist/foo.json` file, which is useful for debugging.
	skips := pipe.SkipMemento{}
	for _, manifest := range ctx.Artifacts.Filter(artifact.ByType(artifact.ScoopManifest)).List() {
		err := doPublish(ctx, manifest, cli)
		if err != nil && pipe.IsSkip(err) {
			skips.Remember(err)
			continue
		}
		if err != nil {
			return err
		}
	}
	return skips.Evaluate()
}

func doPublish(ctx *context.Context, manifest *artifact.Artifact, cl client.Client) error {
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

	repo := client.RepoFromRef(scoop.Repository)
	gpath := path.Join(scoop.Directory, manifest.Name)

	if scoop.Repository.Git.URL != "" {
		return client.NewGitUploadClient(repo.Branch).
			CreateFile(ctx, author, repo, content, gpath, commitMessage)
	}

	cl, err = client.NewIfToken(ctx, cl, scoop.Repository.Token)
	if err != nil {
		return err
	}

	base := client.Repo{
		Name:   scoop.Repository.PullRequest.Base.Name,
		Owner:  scoop.Repository.PullRequest.Base.Owner,
		Branch: scoop.Repository.PullRequest.Base.Branch,
	}

	// try to sync branch
	fscli, ok := cl.(client.ForkSyncer)
	if ok && scoop.Repository.PullRequest.Enabled {
		if err := fscli.SyncFork(ctx, repo, base); err != nil {
			log.WithError(err).Warn("could not sync fork")
		}
	}

	if err := cl.CreateFile(ctx, author, repo, content, gpath, commitMessage); err != nil {
		return err
	}

	if !scoop.Repository.PullRequest.Enabled {
		log.Debug("scoop.pull_request disabled")
		return nil
	}

	log.Info("scoop.pull_request enabled, creating a PR")
	pcl, ok := cl.(client.PullRequestOpener)
	if !ok {
		return fmt.Errorf("client does not support pull requests")
	}

	return pcl.OpenPullRequest(ctx, base, repo, commitMessage, scoop.Repository.PullRequest.Draft)
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

func dataFor(ctx *context.Context, scoop config.Scoop, cl client.ReleaseURLTemplater, artifacts []*artifact.Artifact) (Manifest, error) {
	manifest := Manifest{
		Version:      ctx.Version,
		Architecture: map[string]Resource{},
		Homepage:     scoop.Homepage,
		License:      scoop.License,
		Description:  scoop.Description,
		Persist:      scoop.Persist,
		PreInstall:   scoop.PreInstall,
		PostInstall:  scoop.PostInstall,
		Depends:      scoop.Depends,
		Shortcuts:    scoop.Shortcuts,
	}

	if scoop.URLTemplate == "" {
		url, err := cl.ReleaseURLTemplate(ctx)
		if err != nil {
			return manifest, err
		}
		scoop.URLTemplate = url
	}

	for _, artifact := range artifacts {
		if artifact.Goos != "windows" {
			continue
		}

		var arch string
		switch artifact.Goarch {
		case "386":
			arch = "32bit"
		case "amd64":
			arch = "64bit"
		case "arm64":
			arch = "arm64"
		default:
			continue
		}

		url, err := tmpl.New(ctx).WithArtifact(artifact).Apply(scoop.URLTemplate)
		if err != nil {
			return manifest, err
		}

		sum, err := artifact.Checksum("sha256")
		if err != nil {
			return manifest, err
		}

		log.
			WithField("artifactExtras", artifact.Extra).
			WithField("fromURLTemplate", scoop.URLTemplate).
			WithField("templatedBrewURL", url).
			WithField("sum", sum).
			Debug("scoop url templating")

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
	var result []string
	wrap := artifact.ExtraOr(a, artifact.ExtraWrappedIn, "")
	bins, err := artifact.Extra[[]string](a, artifact.ExtraBinaries)
	if err != nil {
		return nil, err
	}
	for _, b := range bins {
		result = append(result, filepath.Join(wrap, b))
	}
	return result, nil
}
