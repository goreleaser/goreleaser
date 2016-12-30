package release

import (
	"context"
	"log"
	"os"
	"os/exec"

	"github.com/google/go-github/github"
	"github.com/goreleaser/releaser/config"
	"github.com/goreleaser/releaser/split"
	"github.com/goreleaser/releaser/uname"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

type Pipe struct{}

func (Pipe) Name() string {
	return "GithubRelease"
}

func (Pipe) Work(config config.ProjectConfig) error {
	log.Println("Creating release", config.Git.CurrentTag, "on repo", config.Repo, "...")
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.Token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	owner, repo := split.OnSlash(config.Repo)
	r, _, err := client.Repositories.CreateRelease(owner, repo, &github.RepositoryRelease{
		Name:            github.String(config.Git.CurrentTag),
		TagName:         github.String(config.Git.CurrentTag),
		Body:            github.String(description(config.Git.Diff)),
	})
	if err != nil {
		return err
	}
	var g errgroup.Group
	for _, system := range config.Build.Oses {
		for _, arch := range config.Build.Arches {
			system := system
			arch := arch
			g.Go(func() error {
				return upload(client, *r.ID, owner, repo, system, arch, config.BinaryName)
			})
		}
	}
	return g.Wait()
}

func description(diff string) string {
	result := "## Changelog:\n" + diff + "\n\n--\nAutomated with @goreleaser"
	cmd := exec.Command("go", "version")
	bts, err := cmd.CombinedOutput()
	if err != nil {
		return result
	}
	return result + "\nBuilt with " + string(bts)
}

func upload(client *github.Client, releaseID int, owner, repo, system, arch, binaryName string) error {
	name := binaryName + "_" + uname.FromGo(system) + "_" + uname.FromGo(arch) + ".tar.gz"
	file, err := os.Open("dist/" + name)
	if err != nil {
		return err
	}
	defer file.Close()
	log.Println("Uploading", file.Name(), "...")
	_, _, err = client.Repositories.UploadReleaseAsset(owner, repo, releaseID, &github.UploadOptions{
		Name: name,
	}, file)
	return err
}
