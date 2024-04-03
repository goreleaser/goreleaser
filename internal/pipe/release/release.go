package release

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/extrafiles"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// ErrMultipleReleases indicates that multiple releases are defined. ATM only one of them is allowed.
// See https://github.com/goreleaser/goreleaser/pull/809
var ErrMultipleReleases = errors.New("multiple releases are defined. Only one is allowed")

// Pipe for github release.
type Pipe struct{}

func (Pipe) String() string { return "scm releases" }

func (Pipe) Skip(ctx *context.Context) (bool, error) {
	return tmpl.New(ctx).Bool(ctx.Config.Release.Disable)
}

// Default sets the pipe defaults.
func (p Pipe) Default(ctx *context.Context) error {
	if b, _ := p.Skip(ctx); b {
		return pipe.Skip("release is disabled")
	}
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
		if err := setupGitLab(ctx); err != nil {
			return err
		}
	case context.TokenTypeGitea:
		if err := setupGitea(ctx); err != nil {
			return err
		}
	default:
		// We keep github as default for now
		if err := setupGitHub(ctx); err != nil {
			return err
		}
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

func getRepository(ctx *context.Context) (config.Repo, error) {
	repo, err := git.ExtractRepoFromConfig(ctx)
	if err != nil {
		return config.Repo{}, err
	}
	if err := repo.CheckSCM(); err != nil {
		return config.Repo{}, err
	}
	return repo, nil
}

// Publish the release.
func (Pipe) Publish(ctx *context.Context) error {
	c, err := client.New(ctx)
	if err != nil {
		return err
	}
	if err := doPublish(ctx, c); err != nil {
		return err
	}
	log.WithField("url", ctx.ReleaseURL).WithField("published", !ctx.Config.Release.Draft).Info("release created/updated")
	return nil
}

func doPublish(ctx *context.Context, client client.Client) error {
	log.WithField("tag", ctx.Git.CurrentTag).
		WithField("repo", ctx.Config.Release.GitHub.String()).
		Info("creating or updating release")
	if err := ctx.Artifacts.Refresh(); err != nil {
		return err
	}
	body, err := describeBody(ctx)
	if err != nil {
		return err
	}
	releaseID, err := client.CreateRelease(ctx, body.String())
	if err != nil {
		return err
	}

	skipUpload, err := tmpl.New(ctx).Bool(ctx.Config.Release.SkipUpload)
	if err != nil {
		return err
	}
	if skipUpload {
		if err := client.PublishRelease(ctx, releaseID); err != nil {
			return err
		}
		return pipe.Skip("release.skip_upload is set")
	}

	extraFiles, err := extrafiles.Find(ctx, ctx.Config.Release.ExtraFiles)
	if err != nil {
		return err
	}

	for name, path := range extraFiles {
		if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to upload %s: %w", name, err)
		}
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: name,
			Path: path,
			Type: artifact.UploadableFile,
		})
	}

	typeFilters := []artifact.Filter{
		artifact.ByType(artifact.UploadableArchive),
		artifact.ByType(artifact.UploadableBinary),
		artifact.ByType(artifact.UploadableSourceArchive),
		artifact.ByType(artifact.UploadableFile),
		artifact.ByType(artifact.Checksum),
		artifact.ByType(artifact.Signature),
		artifact.ByType(artifact.Certificate),
		artifact.ByType(artifact.LinuxPackage),
		artifact.ByType(artifact.SBOM),
	}
	if ctx.Config.Release.IncludeMeta {
		typeFilters = append(typeFilters, artifact.ByType(artifact.Metadata))
	}
	filters := artifact.Or(typeFilters...)

	if len(ctx.Config.Release.IDs) > 0 {
		filters = artifact.And(filters, artifact.ByIDs(ctx.Config.Release.IDs...))
	}

	g := semerrgroup.New(ctx.Parallelism)
	for _, artifact := range ctx.Artifacts.Filter(filters).List() {
		artifact := artifact
		g.Go(func() error {
			return upload(ctx, client, releaseID, artifact)
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	return client.PublishRelease(ctx, releaseID)
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
