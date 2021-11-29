// Package git provides an integration with the git command
package git

import (
	"bytes"
	"errors"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/shell"
)

// IsRepo returns true if current folder is a git repository.
func IsRepo() bool {
	out, err := Run("rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(out) == "true"
}

// Run runs a git command and returns its output or errors.
func Run(args ...string) (string, error) {
	// TODO: use exex.CommandContext here and refactor.
	baseCmd := []string{
		"git", "-c", "log.showSignature=false",
	}
	cmd := append(baseCmd, args...)
	/* #nosec */

	envs := []string{}
	if env != nil {
		for k, v := range env {
			envs = append(envs, k+"="+v)
		}
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	log.WithField("args", args).Debug("running git")
	err := shell.RunWithOutput("", cmd, envs, stdout, stderr)

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
