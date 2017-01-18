package release

import (
	"log"
	"os"
	"os/exec"

	"github.com/google/go-github/github"
	"github.com/goreleaser/goreleaser/clients"
	"github.com/goreleaser/goreleaser/context"
	"golang.org/x/sync/errgroup"
)

// Pipe for github release
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Releasing to GitHub..."
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	client := clients.GitHub(ctx.Token)

	r, err := getOrCreateRelease(client, ctx)
	if err != nil {
		return err
	}
	var g errgroup.Group
	for _, archive := range ctx.Archives {
		archive := archive
		g.Go(func() error {
			return upload(client, *r.ID, archive, ctx)
		})

	}
	return g.Wait()
}

func getOrCreateRelease(client *github.Client, ctx *context.Context) (*github.RepositoryRelease, error) {
	owner := ctx.ReleaseRepo.Owner
	repo := ctx.ReleaseRepo.Name
	data := &github.RepositoryRelease{
		Name:    github.String(ctx.Git.CurrentTag),
		TagName: github.String(ctx.Git.CurrentTag),
		Body:    github.String(description(ctx.Git.Diff)),
	}
	r, _, err := client.Repositories.GetReleaseByTag(owner, repo, ctx.Git.CurrentTag)
	if err != nil {
		log.Println("Creating release", ctx.Git.CurrentTag, "on", ctx.Config.Release.Repo, "...")
		r, _, err = client.Repositories.CreateRelease(owner, repo, data)
		return r, err
	}
	log.Println("Updating existing release", ctx.Git.CurrentTag, "on", ctx.Config.Release.Repo, "...")
	r, _, err = client.Repositories.EditRelease(owner, repo, *r.ID, data)
	return r, err
}

func description(diff string) string {
	result := "## Changelog\n" + diff + "\n\n--\nAutomated with @goreleaser"
	cmd := exec.Command("go", "version")
	bts, err := cmd.CombinedOutput()
	if err != nil {
		return result
	}
	return result + "\nBuilt with " + string(bts)
}

func upload(client *github.Client, releaseID int, archive string, ctx *context.Context) error {
	archive = archive + "." + ctx.Config.Archive.Format
	file, err := os.Open("dist/" + archive)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	log.Println("Uploading", file.Name(), "...")
	_, _, err = client.Repositories.UploadReleaseAsset(
		ctx.ReleaseRepo.Owner,
		ctx.ReleaseRepo.Name,
		releaseID,
		&github.UploadOptions{Name: archive},
		file,
	)
	return err
}
