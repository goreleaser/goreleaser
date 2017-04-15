package git

import (
	"errors"
	"os/exec"
	"strings"
)

func git(args ...string) (output string, err error) {
	var cmd = exec.Command("git", args...)
	bts, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.New(string(bts))
	}
	return string(bts), err
}

func cleanGit(args ...string) (output string, err error) {
	output, err = git(args...)
	return strings.Replace(strings.Split(output, "\n")[0], "'", "", -1), err
}
