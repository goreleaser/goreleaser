package git

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/pkg/config"
)

// ExtractRepoFromConfig gets the repo name from the Git config.
func ExtractRepoFromConfig() (result config.Repo, err error) {
	if !IsRepo() {
		return result, errors.New("current folder is not a git repository")
	}
	out, err := Run("ls-remote", "--get-url")
	if err != nil {
		return result, fmt.Errorf("no remote configured to list refs from")
	}
	log.WithField("rawurl", out).Debugf("got git url")
	return ExtractRepoFromURL(out)
}

func ExtractRepoFromURL(rawurl string) (config.Repo, error) {
	// removes the .git suffix and any new lines
	s := strings.TrimSuffix(strings.TrimSpace(rawurl), ".git")

	// if the URL contains a :, indicating a SSH config,
	// remove all chars until it, including itself
	// on HTTP and HTTPS URLs it will remove the http(s): prefix,
	// which is ok. On SSH URLs the whole user@server will be removed,
	// which is required.

	// If the url contains more than 1 ':' character, assume we are doing an
	// http URL with a username/password in it, and normalize the URL.
	// Gitlab-CI uses this type of URL
	if strings.Count(s, ":") == 1 {
		s = s[strings.LastIndex(s, ":")+1:]
	}

	// now we can parse it with net/url
	u, err := url.Parse(s)
	if err != nil {
		return config.Repo{}, err
	}

	// split the parsed url path by /, the last parts should be the owner and name
	ss := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")

	// if less than 2 parts, its likely not a valid repository
	if len(ss) < 2 {
		return config.Repo{}, fmt.Errorf("unsupported repository URL: %s", rawurl)
	}
	repo := config.Repo{
		Owner: strings.Join(ss[:len(ss)-1], "/"),
		Name:  ss[len(ss)-1],
	}
	log.WithField("owner", repo.Owner).WithField("name", repo.Name).Debugf("parsed url")
	return repo, nil
}
