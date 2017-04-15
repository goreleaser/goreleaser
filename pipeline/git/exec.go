package git

import (
	"errors"
	"os/exec"
	"strings"
)

func git(pwd string, args ...string) (output string, err error) {
	var allArgs = []string{"-C", pwd}
	allArgs = append(allArgs, args...)
	var cmd = exec.Command("git", allArgs...)
	bts, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.New(string(bts))
	}
	return string(bts), err
}

func cleanGit(pwd string, args ...string) (output string, err error) {
	output, err = git(pwd, args...)
	return strings.Replace(strings.Split(output, "\n")[0], "'", "", -1), err
}
