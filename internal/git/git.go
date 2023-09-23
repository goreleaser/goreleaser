// Package git provides an integration with the git command
package git

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"

	"github.com/caarlos0/log"
)

// IsRepo returns true if current folder is a git repository.
func IsRepo(ctx context.Context) bool {
	out, err := Run(ctx, "rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(out) == "true"
}

func RunWithEnv(ctx context.Context, env []string, args ...string) (string, error) {
	extraArgs := []string{
		"-c", "log.showSignature=false",
	}
	args = append(extraArgs, args...)
	/* #nosec */
	cmd := exec.CommandContext(ctx, "git", args...)

	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(cmd.Env, env...)

	err := cmd.Run()

	log.WithField("args", args).
		WithField("stdout", strings.TrimSpace(stdout.String())).
		WithField("stderr", strings.TrimSpace(stderr.String())).
		Debug("git command result")

	if err != nil {
		return "", errors.New(stderr.String())
	}

	return stdout.String(), nil
}

// Run runs a git command and returns its output or errors.
func Run(ctx context.Context, args ...string) (string, error) {
	return RunWithEnv(ctx, []string{}, args...)
}

// Clean the output.
func Clean(output string, err error) (string, error) {
	output = strings.ReplaceAll(strings.Split(output, "\n")[0], "'", "")
	if err != nil {
		err = errors.New(strings.TrimSuffix(err.Error(), "\n"))
	}
	return output, err
}

// CleanAllLines returns all the non empty lines of the output, cleaned up.
func CleanAllLines(output string, err error) ([]string, error) {
	var result []string
	for _, line := range strings.Split(output, "\n") {
		l := strings.TrimSpace(strings.ReplaceAll(line, "'", ""))
		if l == "" {
			continue
		}
		result = append(result, l)
	}
	// TODO: maybe check for exec.ExitError only?
	if err != nil {
		err = errors.New(strings.TrimSuffix(err.Error(), "\n"))
	}
	return result, err
}
