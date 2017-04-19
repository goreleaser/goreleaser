package defaults

import (
	"errors"
	"os/exec"
	"strings"

	"github.com/goreleaser/goreleaser/config"
)

// remoteRepo gets the repo name from the Git config.
func remoteRepo() (result config.GitHubReleaseRepo, err error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	bts, err := cmd.CombinedOutput()
	// TODO: cover this with tests
	if err != nil {
		return result, errors.New(err.Error() + ": " + string(bts))
	}
	return extractRepoFromURL(string(bts)), nil
}

func extractRepoFromURL(s string) config.GitHubReleaseRepo {
	for _, r := range []string{
		"git@github.com:",
		".git",
		"https://github.com/",
		"\n",
	} {
		s = strings.Replace(s, r, "", -1)
	}
	return toRepo(s)
}

func toRepo(s string) config.GitHubReleaseRepo {
	var ss = strings.Split(s, "/")
	return config.GitHubReleaseRepo{
		Repo: config.Repo{
			Owner: ss[0],
			Name:  ss[1],
		},
	}
}
