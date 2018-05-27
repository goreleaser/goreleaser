package php

import (
	"os"
	"os/exec"
	"strings"

	"github.com/apex/log"
	api "github.com/goreleaser/goreleaser/build"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/pkg/errors"
)

// Default builder instance
var Default = &Builder{}

func init() {
	api.Register("php", Default)
}

// Builder is php builder
type Builder struct{}

// WithDefaults sets the defaults for a php build and returns it
func (*Builder) WithDefaults(build config.Build) config.Build {
	// We re use the config.Build.Main field here
	// An alternative would be to alter the config.Build structure
	// and a a "Composer" property, but i think the `func main()`
	// part reflects nearly the same as the composer.json
	// But then in my head there is a conflict.
	// How to deal with `Gopkg.toml` files (dep package manager)?
	if build.Main == "" {
		build.Main = "composer.json"
	}

	// We add a simply dummy value here
	// to enther the target loop in pipeline/build/build.go
	// to trigger the PHP builder at all
	if len(build.Targets) == 0 {
		build.Targets = []string{"linux_amd64"}
	}

	return build
}

// Build builds a php build
func (*Builder) Build(ctx *context.Context, build config.Build, options api.Options) error {
	if err := checkComposerFile(ctx, build); err != nil {
		return err
	}

	// TODO Make composer binary configurable
	// Some people use a "composer.phar"
	// See https://getcomposer.org/download/
	cmd := []string{"composer", "install"}
	if build.Flags != "" {
		cmd = append(cmd, strings.Fields(build.Flags)...)
	}

	var env = build.Env
	if err := run(ctx, cmd, env); err != nil {
		return errors.Wrapf(err, "failed to build for %s", options.Target)
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.Folder,

		// TODO Make this path configurable
		// For this we would need to extend config.Build
		// but this has right now only Golang specific fields
		Path: ".",
		Name: options.Name,
	})
	return nil
}

func run(ctx *context.Context, command, env []string) error {
	/* #nosec */
	var cmd = exec.CommandContext(ctx, command[0], command[1:]...)
	var log = log.WithField("env", env).WithField("cmd", command)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, env...)
	log.WithField("cmd", command).WithField("env", env).Debug("running")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.WithError(err).Debug("failed")
		return errors.New(string(out))
	}
	return nil
}

func checkComposerFile(ctx *context.Context, build config.Build) error {
	var main = build.Main
	_, err := os.Stat(main)
	if os.IsNotExist(err) {
		log.WithError(err).WithField("composer.json", main).Debug("package manager definition file is missing")
		// composer.json don't exist.
		// We just return the Stat error.
		// if we want to be really nice, we could wrap it
		// in a custom error message, but up to now it
		// fulfills our needs
		return err
	}

	// composer.json seems to exists
	// We could go further here, open the file,
	// parse the json and check if it is valid.
	// But for now this might be enough.

	return nil
}
