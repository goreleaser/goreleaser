package release

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/extrafiles"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// ErrMultipleReleases indicates that multiple releases are defined. ATM only one of them is allowed.
// See https://github.com/goreleaser/goreleaser/pull/809
var ErrMultipleReleases = errors.New("multiple releases are defined. Only one is allowed")

// Pipe for github release.
type Pipe struct{}

func (Pipe) String() string                 { return "scm releases" }
func (Pipe) Skip(ctx *context.Context) bool { return ctx.Config.Release.Disable }

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	numOfReleases := 0
	if ctx.Config.Release.GitHub.String() != "" {
		numOfReleases++
	}
	if ctx.Config.Release.GitLab.String() != "" {
		numOfReleases++
	}
	if ctx.Config.Release.Gitea.String() != "" {
		numOfReleases++
	}
	if numOfReleases > 1 {
		return ErrMultipleReleases
	}

	if ctx.Config.Release.NameTemplate == "" {
		ctx.Config.Release.NameTemplate = "{{.Tag}}"
	}

	switch ctx.TokenType {
	case context.TokenTypeGitLab:
		if ctx.Config.Release.GitLab.Name == "" {
			repo, err := git.ExtractRepoFromConfig()
			if err != nil {
				return err
			}
			ctx.Config.Release.GitLab = repo
		}
		ctx.ReleaseURL = fmt.Sprintf(
			"%s/%s/%s/-/releases/%s",
			ctx.Config.GitLabURLs.Download,
			ctx.Config.Release.GitLab.Owner,
			ctx.Config.Release.GitLab.Name,
			ctx.Git.CurrentTag,
		)
	case context.TokenTypeGitea:
		if ctx.Config.Release.Gitea.Name == "" {
			repo, err := git.ExtractRepoFromConfig()
			if err != nil {
				return err
			}
			ctx.Config.Release.Gitea = repo
		}
		ctx.ReleaseURL = fmt.Sprintf(
			"%s/%s/%s/releases/tag/%s",
			ctx.Config.GiteaURLs.Download,
			ctx.Config.Release.Gitea.Owner,
			ctx.Config.Release.Gitea.Name,
			ctx.Git.CurrentTag,
		)
	default:
		// We keep github as default for now
		if ctx.Config.Release.GitHub.Name == "" {
			repo, err := git.ExtractRepoFromConfig()
			if err != nil && !ctx.Snapshot {
				return err
			}
			ctx.Config.Release.GitHub = repo
		}
		ctx.ReleaseURL = fmt.Sprintf(
			"%s/%s/%s/releases/tag/%s",
			ctx.Config.GitHubURLs.Download,
			ctx.Config.Release.GitHub.Owner,
			ctx.Config.Release.GitHub.Name,
			ctx.Git.CurrentTag,
		)
	}

	// Check if we have to check the git tag for an indicator to mark as pre release
	switch ctx.Config.Release.Prerelease {
	case "auto":
		if ctx.Semver.Prerelease != "" {
			ctx.PreRelease = true
		}
		log.Debugf("pre-release was detected for tag %s: %v", ctx.Git.CurrentTag, ctx.PreRelease)
	case "true":
		ctx.PreRelease = true
	}
	log.Debugf("pre-release for tag %s set to %v", ctx.Git.CurrentTag, ctx.PreRelease)

	return nil
}

// Publish the release.
func (Pipe) Publish(ctx *context.Context) error {
	c, err := client.New(ctx)
	if err != nil {
		return err
	}
	return doPublish(ctx, c)
}

func doPublish(ctx *context.Context, client client.Client) error {
	log.WithField("tag", ctx.Git.CurrentTag).
		WithField("repo", ctx.Config.Release.GitHub.String()).
		Info("creating or updating release")
	body, err := describeBody(ctx)
	if err != nil {
		return err
	}
	releaseID, err := client.CreateRelease(ctx, body.String())
	if err != nil {
		return err
	}

	extraFiles, err := extrafiles.Find(ctx, ctx.Config.Release.ExtraFiles)
	if err != nil {
		return err
	}

	for name, path := range extraFiles {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("failed to upload %s: %w", name, err)
		}
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: name,
			Path: path,
			Type: artifact.UploadableFile,
		})
	}

	filters := artifact.Or(
		artifact.ByType(artifact.UploadableArchive),
		artifact.ByType(artifact.UploadableBinary),
		artifact.ByType(artifact.UploadableSourceArchive),
		artifact.ByType(artifact.Checksum),
		artifact.ByType(artifact.Signature),
		artifact.ByType(artifact.Certificate),
		artifact.ByType(artifact.LinuxPackage),
		artifact.ByType(artifact.SBOM),
	)

	if len(ctx.Config.Release.IDs) > 0 {
		filters = artifact.And(filters, artifact.ByIDs(ctx.Config.Release.IDs...))
	}

	filters = artifact.Or(filters, artifact.ByType(artifact.UploadableFile))

	g := semerrgroup.New(ctx.Parallelism)
	for _, artifact := range ctx.Artifacts.Filter(filters).List() {
		artifact := artifact
		g.Go(func() error {
			return upload(ctx, client, releaseID, artifact)
		})
	}
	return g.Wait()
}

func upload(ctx *context.Context, cli client.Client, releaseID string, artifact *artifact.Artifact) error {
	var try int
	tryUpload := func() error {
		try++
		file, err := os.Open(artifact.Path)
		if err != nil {
			return err
		}
		defer file.Close()
		log.WithField("file", file.Name()).WithField("name", artifact.Name).Info("uploading to release")
		if err := cli.Upload(ctx, releaseID, artifact, file); err != nil {
			log.WithField("try", try).
				WithField("artifact", artifact.Name).
				WithError(err).
				Warnf("failed to upload artifact, will retry")
			return err
		}
		return nil
	}

	var err error
	for try < 10 {
		err = tryUpload()
		if err == nil {
			return nil
		}
		if errors.As(err, &client.RetriableError{}) {
			time.Sleep(time.Duration(try*50) * time.Millisecond)
			continue
		}
		break
	}

	return fmt.Errorf("failed to upload %s after %d tries: %w", artifact.Name, try, err)
}
