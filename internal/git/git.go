// Package git provides an integration with the git command
package git

import (
	"errors"
	"os/exec"
	"strings"
)

// IsRepo returns true if current folder is a git repository
func IsRepo() bool {
	out, err := Run("rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(out) == "true"
}

// Run runs a git command and returns its output or errors
func Run(args ...string) (output string, err error) {
	/* #nosec */
	var cmd = exec.Command("git", args...)
	bts, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.New(string(bts))
	}
	return string(bts), err
}

// Clean the output
func Clean(output string, err error) (string, error) {
	return strings.Replace(strings.Split(output, "\n")[0], "'", "", -1), err
}
