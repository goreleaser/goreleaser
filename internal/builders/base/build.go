// Package base contains common functions for all builders.
package base

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gio"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
)

type options struct {
	allowBuildMode bool
}

type ValidateOption func(o *options)

func WithBuildMode(o *options) { o.allowBuildMode = true }

// ValidateNonGoConfig makes sure that Go-specific configurations are not set.
func ValidateNonGoConfig(build config.Build, opts ...ValidateOption) error {
	var o options
	for _, v := range opts {
		v(&o)
	}

	if len(build.Ldflags) > 0 {
		return errors.New("ldflags is not used for " + build.Builder)
	}

	if len(slices.Concat(
		build.Goos,
		build.Goarch,
		build.Goamd64,
		build.Go386,
		build.Goarm,
		build.Goarm64,
		build.Gomips,
		build.Goppc64,
		build.Goriscv64,
	)) > 0 {
		return fmt.Errorf("all go* fields are not used for %s, set targets instead", build.Builder)
	}

	if len(build.Ignore) > 0 {
		return fmt.Errorf("ignore is not used for %s, set targets instead", build.Builder)
	}

	if build.Buildmode != "" && !o.allowBuildMode {
		return errors.New("buildmode is not used for " + build.Builder)
	}

	if len(build.Tags) > 0 {
		return errors.New("tags is not used for " + build.Builder)
	}

	if len(build.Asmflags) > 0 {
		return errors.New("asmflags is not used for " + build.Builder)
	}

	if len(build.BuildDetailsOverrides) > 0 {
		return errors.New("overrides is not used for " + build.Builder)
	}

	return nil
}

// ChTimes sets the mod time for the artifact path, if a mod timestamp is set
// in the build.
func ChTimes(build config.Build, tpl *tmpl.Template, a *artifact.Artifact) error {
	modTimestamp, err := tpl.Apply(build.ModTimestamp)
	if err != nil {
		return err
	}
	if modTimestamp == "" {
		return nil
	}
	if err := gio.Chtimes(a.Path, modTimestamp); err != nil {
		return err
	}
	return nil
}

// TemplateEnv templates the build.Env and returns it.
func TemplateEnv(input []string, tpl *tmpl.Template) ([]string, error) {
	var output []string
	for _, e := range input {
		ee, err := tpl.Apply(e)
		if err != nil {
			return nil, err
		}
		log.Debugf("env %q evaluated to %q", e, ee)
		if ee != "" {
			output = append(output, ee)
			tpl = tpl.SetEnv(ee)
		}
	}
	return output, nil
}

// Exec executes the given command with the given env in the given dir,
// handling output and errors.
func Exec(ctx context.Context, command []string, env []string, dir string) error {
	/* #nosec */
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Env = env
	cmd.Dir = dir
	log.WithField("cmd", command[0]).Debug("executing")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	if s := string(out); s != "" {
		log.WithField("output", s).Info(command[0] + " " + command[1])
	}
	return nil
}
