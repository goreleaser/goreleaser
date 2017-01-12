package git

import (
	"errors"
	"os/exec"
	"strings"
)

func RemoteRepoName() (result string, err error) {
	cmd := exec.Command("git", "remote", "get-url", "origin", "--push")
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
