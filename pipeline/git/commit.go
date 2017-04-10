package git

import (
	"errors"
	"os/exec"
	"strings"
)

func commitHash() (string, error) {
	cmd := exec.Command(
		"git",
		"show",
		"--format='%H'",
		"HEAD",
	)
	bts, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.New(err.Error() + ": " + string(bts))
	}
	return strings.Replace(strings.Split(string(bts), "\n")[0], "'", "", -1), err
}
