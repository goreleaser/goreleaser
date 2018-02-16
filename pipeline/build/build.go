// Package build provides a pipe that can build binaries for several
// languages.
package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	builders "github.com/goreleaser/goreleaser/build"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"

	// langs to init
	_ "github.com/goreleaser/goreleaser/internal/builders/golang"
	"time"
	"bytes"
	"text/template"
)

// Pipe for build
type Pipe struct{}

func (Pipe) String() string {
	return "building binaries"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	for _, build := range ctx.Config.Builds {
		log.WithField("build", build).Debug("building")
		if err := runPipeOnBuild(ctx, build); err != nil {
			return err
		}
	}
	return nil
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	for i, build := range ctx.Config.Builds {
		ctx.Config.Builds[i] = buildWithDefaults(ctx, build)
	}
	if len(ctx.Config.Builds) == 0 {
		ctx.Config.Builds = []config.Build{
			buildWithDefaults(ctx, ctx.Config.SingleBuild),
		}
	}
	return nil
}

func buildWithDefaults(ctx *context.Context, build config.Build) config.Build {
	if build.Lang == "" {
		build.Lang = "go"
	}
	if build.Binary == "" {
		build.Binary = ctx.Config.Release.GitHub.Name
	}
	for k, v := range build.Env {
		build.Env[k] = os.ExpandEnv(v)
	}
	return builders.For(build.Lang).WithDefaults(build)
}

func runPipeOnBuild(ctx *context.Context, build config.Build) error {
	if err := runHook(ctx, build.Env, build.Hooks.Pre); err != nil {
		return errors.Wrap(err, "pre hook failed")
	}
	sem := make(chan bool, ctx.Parallelism)
	var g errgroup.Group
	for _, target := range build.Targets {
		sem <- true
		target := target
		build := build
		g.Go(func() error {
			defer func() {
				<-sem
			}()
			return doBuild(ctx, build, target)
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return errors.Wrap(runHook(ctx, build.Env, build.Hooks.Post), "post hook failed")
}

func runHook(ctx *context.Context, env []string, hook string) error {
	if hook == "" {
		return nil
	}
	log.WithField("hook", hook).Info("running hook")
	cmd := strings.Fields(hook)
	return run(ctx, cmd, env)
}

func doBuild(ctx *context.Context, build config.Build, target string) error {
	var ext = extFor(target)

	binary, err := binary(ctx, build)
	if err != nil {
		return err
	}

	build.Binary = binary
	var name = build.Binary + ext
	var path = filepath.Join(ctx.Config.Dist, target, name)
	log.WithField("binary", path).Info("building")
	return builders.For(build.Lang).Build(ctx, build, builders.Options{
		Target: target,
		Name:   name,
		Path:   path,
		Ext:    ext,
	})
}

func binary(ctx *context.Context, build config.Build) (string, error) {
	var data = struct {
		Commit  string
		Tag     string
		Version string
		Date    string
		Env     map[string]string
	}{
		Commit:  ctx.Git.Commit,
		Tag:     ctx.Git.CurrentTag,
		Version: ctx.Version,
		Date:    time.Now().UTC().Format(time.RFC3339),
		Env:     ctx.Env,
	}
	var out bytes.Buffer
	t, err := template.New("binary").
		Option("missingkey=error").
		Parse(build.Binary)
	if err != nil {
		return "", err
	}
	err = t.Execute(&out, data)
	return out.String(), err
}

func extFor(target string) string {
	if strings.Contains(target, "windows") {
		return ".exe"
	}
	return ""
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
