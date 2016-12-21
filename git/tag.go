package git

import (
	"os/exec"
	"strings"
)

func CurrentTag() (tag string, err error) {
	return getTag("master")
}

func PreviousTag() (tag string, err error) {
	current, err := CurrentTag()
	if err != nil {
		return tag, err
	}
	return getTag(current + "^")
}

func getTag(ref string) (tag string, err error) {
	cmd := exec.Command(
		"git",
		"describe",
		"--tags",
		"--abbrev=0",
		"--always",
		ref,
	)
	bts, err := cmd.CombinedOutput()
	if err != nil {
		return tag, err
	}
	return strings.Split(string(bts), "\n")[0], err
}
