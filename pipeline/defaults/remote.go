package defaults

import (
	"errors"
	"os/exec"
	"strings"
)

// remoteRepo gets the repo name from the Git config.
func remoteRepo() (result string, err error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	bts, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.New(err.Error() + ": " + string(bts))
	}
	return extractRepoFromURL(string(bts)), nil
}

func extractRepoFromURL(s string) string {
	for _, r := range []string{
		"git@github.com:",
		".git",
		"https://github.com/",
		"\n",
	} {
		s = strings.Replace(s, r, "", -1)
	}
	return s
}
