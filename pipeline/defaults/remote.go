package defaults

import (
	"errors"
	"os/exec"
	"strings"

	"github.com/goreleaser/goreleaser/config"
)

// remoteRepo gets the repo name from the Git config.
func remoteRepo() (result config.Repo, err error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	bts, err := cmd.CombinedOutput()
	if err != nil {
		return result, errors.New(err.Error() + ": " + string(bts))
	}
	return extractRepoFromURL(string(bts)), nil
}

func extractRepoFromURL(s string) config.Repo {
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

func toRepo(s string) config.Repo {
	var ss = strings.Split(s, "/")
	return config.Repo{
		Owner: ss[0],
		Name:  ss[1],
	}
}
