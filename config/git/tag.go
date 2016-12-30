package git

import (
	"errors"
	"os/exec"
	"strings"
)

// CurrentTag tag being built
func CurrentTag() (tag string, err error) {
	return getTag("")
}

// PreviousTag previous tag of the base tag
func PreviousTag(base string) (tag string, err error) {
	return getTag(base + "^")
}

func getTag(ref string) (tag string, err error) {
	cmd := exec.Command(
		"git",
		"describe",
		"--tags",
		"--abbrev=0",
		"--always",
	)
	if ref != "" {
		cmd.Args = append(cmd.Args, ref)
	}
	bts, err := cmd.CombinedOutput()
	if err != nil {
		return tag, errors.New(err.Error() + ": " + string(bts))
	}
	return strings.Split(string(bts), "\n")[0], err
}
