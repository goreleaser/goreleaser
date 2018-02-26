// Package git provides an integration with the git command
package git

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"

	"github.com/apex/log"
)

// IsRepo returns true if current folder is a git repository
func IsRepo() bool {
	out, err := Run("rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(out) == "true"
}

// Run runs a git command and returns its output or errors
func Run(args ...string) (string, error) {
	/* #nosec */
	var cmd = exec.Command("git", args...)
	log.WithField("args", args).Debug("running git")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", errors.New(stderr.String())
	}
	log.WithField("output", stdout.String()).Debug("git result")
	return stdout.String(), nil
}

// Clean the output
func Clean(output string, err error) (string, error) {
	output = strings.Replace(strings.Split(output, "\n")[0], "'", "", -1)
	if err != nil {
		err = errors.New(strings.TrimSuffix(err.Error(), "\n"))
	}
	return output, err
}
