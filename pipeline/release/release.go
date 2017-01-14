package release

import (
	"log"
	"os"
	"os/exec"

	"github.com/google/go-github/github"
	"github.com/goreleaser/releaser/context"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

// Pipe for github release
type Pipe struct{}

// Name of the pipe
func (Pipe) Name() string {
	return "GithubRelease"
}

// Run the pipe
func (Pipe) Run(context *context.Context) error {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: context.Token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	r, err := getOrCreateRelease(client, context)
	if err != nil {
		return err
	}
	var g errgroup.Group
	for _, archive := range context.Archives {
		archive := archive
		g.Go(func() error {
			return upload(client, *r.ID, archive, context)
		})

	}
	return g.Wait()
}

func getOrCreateRelease(client *github.Client, context *context.Context) (*github.RepositoryRelease, error) {
	owner := context.Repo.Owner
	repo := context.Repo.Name
	data := &github.RepositoryRelease{
		Name:    github.String(context.Git.CurrentTag),
		TagName: github.String(context.Git.CurrentTag),
		Body:    github.String(description(context.Git.Diff)),
	}
	r, res, err := client.Repositories.GetReleaseByTag(owner, repo, context.Git.CurrentTag)
	if err != nil && res.StatusCode == 404 {
		log.Println("Creating release", config.Git.CurrentTag, "on", context.Config.Repo, "...")
		r, _, err = client.Repositories.CreateRelease(owner, repo, data)
		return r, err
	}
	log.Println("Updating existing release", config.Git.CurrentTag, "on", context.Config.Repo, "...")
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

func upload(client *github.Client, releaseID int, archive string, context *context.Context) error {
	archive = archive + "." + context.Config.Archive.Format
	file, err := os.Open("dist/" + archive)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	log.Println("Uploading", file.Name(), "...")
	_, _, err = client.Repositories.UploadReleaseAsset(
		context.Repo.Owner,
		context.Repo.Name,
		releaseID,
		&github.UploadOptions{Name: archive},
		file,
	)
	return err
}
