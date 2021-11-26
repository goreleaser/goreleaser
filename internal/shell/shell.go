package shell

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"

	"github.com/apex/log"

	"github.com/goreleaser/goreleaser/internal/logext"
)

// Run a shell command with given arguments and envs
func Run(dir string, command, env []string) error {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	return RunWithOutput(dir, command, env, stdout, stderr)
}

// RunWithOutput is the same as Run but receives the stdout and stderr fo output
func RunWithOutput(dir string, command, env []string, stdout, stderr *bytes.Buffer) error {
	fields := log.Fields{
		"cmd": command,
		"env": env,
	}

	/* #nosec */
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = env

	cmd.Stderr = io.MultiWriter(logext.NewWriter(fields, logext.Error), stderr)
	cmd.Stdout = io.MultiWriter(logext.NewWriter(fields, logext.Info), stdout)

	if dir != "" {
		cmd.Dir = dir
	}

	log.WithFields(fields).Debug("running")
	if err := cmd.Run(); err != nil {
		log.WithFields(fields).WithError(err).Debug("failed")
		return fmt.Errorf("%q: %w", stdout.String(), err)
	}

	return nil
}
