package release

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"github.com/goreleaser/releaser/config"
	"github.com/goreleaser/releaser/uname"
	"golang.org/x/oauth2"
	"os/exec"
)

func Release(version, diff string, config config.ProjectConfig) error {
	fmt.Println("Creating release", version, "on repo", config.Repo, "...")
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.Token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	owner := strings.Split(config.Repo, "/")[0]
	repo := strings.Split(config.Repo, "/")[1]
	releaseData := &github.RepositoryRelease{
		Name:            github.String(version),
		TagName:         github.String(version),
		Body:            github.String(description(diff)),
	}
	r, _, err := client.Repositories.CreateRelease(owner, repo, releaseData)
	if err != nil {
		return err
	}
	for _, system := range config.Build.Oses {
		for _, arch := range config.Build.Arches {
			if err := upload(client, *r.ID, owner, repo, system, arch, config.BinaryName); err != nil {
				return err
			}
		}
	}
	return nil
}

func description(diff string) string {
	result := diff + "\n\nAutomated with @goreleaser"
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
	fmt.Println("Uploading", file.Name(), "...")
	_, _, err = client.Repositories.UploadReleaseAsset(owner, repo, releaseID, &github.UploadOptions{
		Name: name,
	}, file)
	return err
}
