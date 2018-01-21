package build

import (
	"errors"
	"os"
	"os/exec"
	"sync"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/build/buildtarget"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
)

var (
	builders = map[string]Builder{}
	lock     sync.Mutex
)

func Register(lang string, builder Builder) {
	lock.Lock()
	builders[lang] = builder
	lock.Unlock()
}

func For(lang string) Builder {
	return builders[lang]
}

type Options struct {
	Target          buildtarget.Target
	Name, Path, Ext string
}

type Builder interface {
	Build(ctx *context.Context, build config.Build, options Options) error
}

func Run(ctx *context.Context, command, env []string) error {
	/* #nosec */
	var cmd = exec.CommandContext(ctx, command[0], command[1:]...)
	var log = log.WithField("env", env).WithField("cmd", command)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, env...)
	log.Debug("running")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.WithError(err).Debug("failed")
		return errors.New(string(out))
	}
	return nil
}
