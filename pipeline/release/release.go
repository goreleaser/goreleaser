package release

import (
	"log"
	"os"
	"path/filepath"

	"github.com/goreleaser/goreleaser/clients"
	"github.com/goreleaser/goreleaser/context"
	"golang.org/x/sync/errgroup"
)

// Pipe for github release
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Releasing to GitHub"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	client := clients.NewGitHubClient(ctx)
	return doRun(ctx, client)
}

func doRun(ctx *context.Context, client clients.Client) error {
	log.Println("Creating or updating release", ctx.Git.CurrentTag, "on", ctx.Config.Release.GitHub.String())
	releaseID, err := client.CreateRelease(ctx)
	if err != nil {
		return err
	}
	var g errgroup.Group
	for _, archive := range ctx.Archives {
		archive := archive
		g.Go(func() error {
			return upload(ctx, client, releaseID, archive, ctx.Config.Archive.Format)
		})
		for _, format := range ctx.Config.FPM.Formats {
			format := format
			g.Go(func() error {
				return upload(ctx, client, releaseID, archive, format)
			})
		}
	}
	return g.Wait()
}

func upload(ctx *context.Context, client clients.Client, releaseID int, archive, format string) error {
	archive = archive + "." + format
	var path = filepath.Join(ctx.Config.Dist, archive)
	// In case the file doesn't exist, we just ignore it.
	// We do this because we can get invalid combinations of archive+format here,
	// like darwinamd64 + deb or something like that.
	// It's assumed that the archive pipe would fail the entire thing in case it fails to
	// generate some archive, as well fpm pipe is expected to fail if something wrong happens.
	// So, here, we just assume IsNotExist as an expected error.
	// TODO: maybe add a list of files to upload in the context so we don't have to do this.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	log.Println("Uploading", file.Name())
	return client.Upload(ctx, releaseID, archive, file)
}
