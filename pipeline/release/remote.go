package release

import (
	"strings"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/pkg/errors"
)

// remoteRepo gets the repo name from the Git config.
func remoteRepo() (result config.Repo, err error) {
	if !git.IsRepo() {
		return result, errors.New("current folder is not a git repository")
	}
	out, err := git.Run("config", "--get", "remote.origin.url")
	if err != nil {
		return result, errors.Wrap(err, "repository doesn't have an `origin` remote")
	}
	return extractRepoFromURL(out), nil
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
