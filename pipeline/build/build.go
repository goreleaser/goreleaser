// Package build implements Pipe and can build Go projects for
// several platforms, with pre and post hook support.
package build

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/buildtarget"
	"github.com/goreleaser/goreleaser/internal/ext"
	"github.com/goreleaser/goreleaser/internal/name"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// Pipe for build
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Building binaries"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	for _, build := range ctx.Config.Builds {
		log.WithField("build", build).Debug("building")
		if err := checkMain(ctx, build); err != nil {
			return err
		}
		if err := runPipeOnBuild(ctx, build); err != nil {
			return err
		}
	}
	return nil
}

func checkMain(ctx *context.Context, build config.Build) error {
	if strings.HasSuffix(build.Main, ".go") {
		file, err := parser.ParseFile(token.NewFileSet(), build.Main, nil, 0)
		if err != nil {
			return errors.Wrapf(err, "failed to parse file: %s", build.Main)
		}
		if !hasMain(file) {
			return fmt.Errorf("build for %s does not contain a main function", build.Binary)
		}
		return nil
	}
	var dir = build.Main
	if dir == "" {
		dir = "."
	}
	packs, err := parser.ParseDir(token.NewFileSet(), dir, nil, 0)
	if err != nil {
		return errors.Wrapf(err, "failed to parse dir: %s", dir)
	}
	for _, pack := range packs {
		for _, file := range pack.Files {
			if hasMain(file) {
				return nil
			}
		}
	}
	return fmt.Errorf("build for %s does not contain a main function", build.Binary)
}

func hasMain(file *ast.File) bool {
	for _, decl := range file.Decls {
		fn, isFn := decl.(*ast.FuncDecl)
		if !isFn {
			continue
		}
		if fn.Name.Name == "main" && fn.Recv == nil {
			return true
		}
	}
	return false
}

func runPipeOnBuild(ctx *context.Context, build config.Build) error {
	if err := runHook(build.Env, build.Hooks.Pre); err != nil {
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
	return errors.Wrap(runHook(build.Env, build.Hooks.Post), "post hook failed")
}

func runHook(env []string, hook string) error {
	if hook == "" {
		return nil
	}
	log.WithField("hook", hook).Info("running hook")
	cmd := strings.Fields(hook)
	return run(buildtarget.Runtime, cmd, env)
}

func doBuild(ctx *context.Context, build config.Build, target buildtarget.Target) error {
	folder, err := name.For(ctx, target)
	if err != nil {
		return err
	}
	var binaryName = build.Binary + ext.For(target)
	var prettyName = binaryName
	if ctx.Config.Archive.Format == "binary" {
		binaryName, err = name.ForBuild(ctx, build, target)
		if err != nil {
			return err
		}
		binaryName = binaryName + ext.For(target)
	}
	var binary = filepath.Join(ctx.Config.Dist, folder, binaryName)
	ctx.AddBinary(target.String(), folder, prettyName, binary)
	log.WithField("binary", binary).Info("building")
	cmd := []string{"go", "build"}
	if build.Flags != "" {
		cmd = append(cmd, strings.Fields(build.Flags)...)
	}
	flags, err := ldflags(ctx, build)
	if err != nil {
		return err
	}
	cmd = append(cmd, "-ldflags="+flags, "-o", binary, build.Main)
	return errors.Wrapf(run(target, cmd, build.Env), "failed to build for %s", target)
}

func run(target buildtarget.Target, command, env []string) error {
	var cmd = exec.Command(command[0], command[1:]...)
	env = append(env, target.Env()...)
	var log = log.WithField("target", target.PrettyString()).
		WithField("env", env).
		WithField("cmd", command)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, env...)
	log.Debug("running")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.WithError(err).Debug("failed")
		return errors.New(string(out))
	}
	return nil
}
