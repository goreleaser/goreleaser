package git

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/pkg/config"
)

// ExtractRepoFromConfig gets the repo name from the Git config.
func ExtractRepoFromConfig(ctx context.Context) (result config.Repo, err error) {
	if !IsRepo(ctx) {
		return result, errors.New("current folder is not a git repository")
	}
	out, err := Clean(Run(ctx, "ls-remote", "--get-url"))
	if err != nil {
		return result, fmt.Errorf("no remote configured to list refs from")
	}
	// This is a relative remote URL and requires some additional processing
	if out == "." {
		return extractRelativeRepoFromConfig(ctx)
	}
	log.WithField("rawurl", out).Debugf("got git url")
	return ExtractRepoFromURL(out)
}

func extractRelativeRepoFromConfig(ctx context.Context) (result config.Repo, err error) {
	out, err := Clean(Run(ctx, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"))
	if err != nil || out == "" {
		return result, fmt.Errorf("unable to get upstream while qualifying relative remote")
	}
	out, err = Clean(Run(ctx, "config", "--get", fmt.Sprintf("branch.%s.remote", out)))
	if err != nil || out == "" {
		return result, fmt.Errorf("unable to get upstream's remote while qualifying relative remote")
	}
	out, err = Clean(Run(ctx, "ls-remote", "--get-url", out))
	if err != nil {
		return result, fmt.Errorf("unable to get upstream while qualifying relative remote")
	}
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
		return config.Repo{
			RawURL: rawurl,
		}, err
	}

	// split the parsed url path by /, the last parts should be the owner and name
	ss := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")

	// if empty, returns an error
	if len(ss) == 0 || ss[0] == "" {
		return config.Repo{
			RawURL: rawurl,
		}, fmt.Errorf("unsupported repository URL: %s", rawurl)
	}

	// if less than 2 parts, its likely not a valid repository, but we'll allow it.
	if len(ss) < 2 {
		return config.Repo{
			RawURL: rawurl,
			Owner:  ss[0],
		}, nil
	}
	repo := config.Repo{
		RawURL: rawurl,
		Owner:  path.Join(ss[:len(ss)-1]...),
		Name:   ss[len(ss)-1],
	}
	log.WithField("owner", repo.Owner).WithField("name", repo.Name).Debugf("parsed url")
	return repo, nil
}
