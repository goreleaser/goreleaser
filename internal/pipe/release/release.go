package release

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/extrafiles"
	"github.com/goreleaser/goreleaser/v2/internal/git"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
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
	if !ctx.Config.Release.Draft {
		log.WithField("url", ctx.ReleaseURL).
			Info("release published")
	}
	return nil
}

func doPublish(ctx *context.Context, client client.Client) error {
	log.WithField("tag", ctx.Git.CurrentTag).
		WithField("repo", ctx.Config.Release.GitHub.String()).
		Info("releasing")
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

	types := []artifact.Type{
		artifact.UploadableArchive,
		artifact.UploadableBinary,
		artifact.UploadableSourceArchive,
		artifact.Makeself,
		artifact.UploadableFile,
		artifact.Checksum,
		artifact.Signature,
		artifact.Certificate,
		artifact.LinuxPackage,
		artifact.SBOM,
		artifact.PyWheel,
		artifact.PySdist,
	}
	if ctx.Config.Release.IncludeMeta {
		types = append(types, artifact.Metadata)
	}
	filters := artifact.And(
		artifact.ByTypes(types...),
		artifact.ByIDs(ctx.Config.Release.IDs...),
	)

	g := semerrgroup.New(ctx.Parallelism)
	for _, artifact := range ctx.Artifacts.Filter(filters).List() {
		g.Go(func() error {
			if err := upload(ctx, client, releaseID, artifact); err != nil {
				return fmt.Errorf("failed to upload %s: %w", artifact.Name, err)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	return client.PublishRelease(ctx, releaseID)
}

func upload(ctx *context.Context, cli client.Client, releaseID string, artifact *artifact.Artifact) error {
	return retry.Do(
		func() error {
			log.WithField("file", artifact.Path).
				WithField("name", artifact.Name).
				Info("uploading to release")
			file, err := os.Open(artifact.Path)
			if err != nil {
				return err
			}
			defer file.Close()
			return cli.Upload(ctx, releaseID, artifact, file)
		},
		retry.Attempts(10),
		retry.Delay(50*time.Millisecond),
		retry.LastErrorOnly(true),
		retry.RetryIf(func(err error) bool { return errors.As(err, &client.RetriableError{}) }),
	)
}
