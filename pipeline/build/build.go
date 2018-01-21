// Package build needs to be documented
package build

import (
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	builders "github.com/goreleaser/goreleaser/build"
	"github.com/goreleaser/goreleaser/build/buildtarget"
	"github.com/goreleaser/goreleaser/build/ext"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"

	// langs to init
	_ "github.com/goreleaser/goreleaser/internal/builders/golang"
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
	if build.Main == "" {
		build.Main = "."
	}
	if len(build.Goos) == 0 {
		build.Goos = []string{"linux", "darwin"}
	}
	if len(build.Goarch) == 0 {
		build.Goarch = []string{"amd64", "386"}
	}
	if len(build.Goarm) == 0 {
		build.Goarm = []string{"6"}
	}
	if build.Ldflags == "" {
		build.Ldflags = "-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}"
	}
	return build
}

func runPipeOnBuild(ctx *context.Context, build config.Build) error {
	if err := runHook(ctx, build.Env, build.Hooks.Pre); err != nil {
		return errors.Wrap(err, "pre hook failed")
	}
	sem := make(chan bool, ctx.Parallelism)
	var g errgroup.Group
	for _, target := range buildtarget.All(build) {
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
	return builders.Run(ctx, buildtarget.Runtime, cmd, env)
}

func doBuild(ctx *context.Context, build config.Build, target buildtarget.Target) error {
	var ext = ext.For(target)
	var binaryName = build.Binary + ext
	var binary = filepath.Join(ctx.Config.Dist, target.String(), binaryName)
	log.WithField("binary", binary).Info("building")
	return builders.For(build.Lang).Build(ctx, build, builders.Options{
		Target: target,
		Name:   binaryName,
		Path:   binary,
		Ext:    ext,
	})
}
