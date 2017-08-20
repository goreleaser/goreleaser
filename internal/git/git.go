// Package git provides an integration with the git command
package git

import (
	"errors"
	"os/exec"
)

// Run runs a git command and returns its output or errors
func Run(args ...string) (output string, err error) {
	var cmd = exec.Command("git", args...)
	bts, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.New(string(bts))
	}
	return string(bts), err
}
