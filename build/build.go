// Package build provides the API for external builders
package build

import (
	"errors"
	"os"
	"os/exec"
	"sync"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
)

var (
	builders = map[string]Builder{}
	lock     sync.Mutex
)

// Register register a builder to a given lang
func Register(lang string, builder Builder) {
	lock.Lock()
	builders[lang] = builder
	lock.Unlock()
}

// For gets the previously register builder for the given lang
func For(lang string) Builder {
	return builders[lang]
}

// Options to be passed down to a builder
type Options struct {
	Name, Path, Ext, Target string
}

// Builder defines a builder
type Builder interface {
	Default(build config.Build) config.Build
	Build(ctx *context.Context, build config.Build, options Options) error
}

// Run runs a command within the given context and env
func Run(ctx *context.Context, command, env []string) error {
	/* #nosec */
	var cmd = exec.CommandContext(ctx, command[0], command[1:]...)
	var log = log.WithField("env", env).WithField("cmd", command)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, env...)
	// TODO: improve debug here
	log.Debug("running")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.WithError(err).Debug("failed")
		return errors.New(string(out))
	}
	return nil
}
