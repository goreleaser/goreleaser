// Package git provides an integration with the git command
package git

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"

	"github.com/apex/log"
)

// IsRepo returns true if current folder is a git repository.
func IsRepo() bool {
	out, err := Run("rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(out) == "true"
}

// Run runs a git command and returns its output or errors.
func Run(args ...string) (string, error) {
	// TODO: use exex.CommandContext here and refactor.
	extraArgs := []string{
		"-c", "log.showSignature=false",
	}
	args = append(extraArgs, args...)
	/* #nosec */
	cmd := exec.Command("git", args...)

	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.WithField("args", args).Debug("running git")
	err := cmd.Run()

	log.WithField("stdout", stdout.String()).
		WithField("stderr", stderr.String()).
		Debug("git result")

	if err != nil {
		return "", errors.New(stderr.String())
	}

	return stdout.String(), nil
}

// Clean the output.
func Clean(output string, err error) (string, error) {
	output = strings.ReplaceAll(strings.Split(output, "\n")[0], "'", "")
	if err != nil {
		err = errors.New(strings.TrimSuffix(err.Error(), "\n"))
	}
	return output, err
}
