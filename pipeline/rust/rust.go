package rust

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pipeline"
)

// Pipe for rust
type Pipe struct{}

func (Pipe) String() string {
	return "building rust binaries"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	if len(ctx.Config.Rust) == 0 {
		return pipeline.Skip("rust section is not configured")
	}

	for _, rust := range ctx.Config.Rust {
		log.WithField("rust", rust).Debug("building")
		if err := runPipeOnBuild(ctx, rust); err != nil {
			return err
		}
	}
	return nil
}

/*
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
*/
/*
func buildWithDefaults(ctx *context.Context, build config.Build) config.Build {
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
*/

func runPipeOnBuild(ctx *context.Context, rust config.Rust) error {
	if err := runHook(ctx, rust.Env, rust.Hooks.Pre); err != nil {
		return errors.Wrap(err, "pre hook failed")
	}

	sem := make(chan bool, ctx.Parallelism)
	var g errgroup.Group
	for _, target := range rust.Target {
		sem <- true
		target := target
		rust := rust
		g.Go(func() error {
			defer func() {
				<-sem
			}()
			return doBuild(ctx, rust, target)
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	return errors.Wrap(runHook(ctx, rust.Env, rust.Hooks.Post), "post hook failed")
}

func runHook(ctx *context.Context, env []string, hook string) error {
	if hook == "" {
		return nil
	}
	log.WithField("hook", hook).Info("running hook")
	cmd := strings.Fields(hook)
	// Normally we would include "buildtarget.Runtime"
	// as a second argument here.
	// But this is not supported in the Rust pipe
	return run(ctx, "", cmd, env)
}

func doBuild(ctx *context.Context, build config.Rust, target string) error {
	//var ext = ext.For(target)
	// TODO Support for windows (exe)
	var ext = ""
	var binaryName = build.Binary + ext
	var binary = filepath.Join(ctx.Config.Dist, target, binaryName)
	log.WithField("binary", binary).Info("building")
	cmd := []string{"cargo", "build"}
	cmd = append(cmd, "--release", "--bin", binaryName, "--target", target)
	if err := run(ctx, target, cmd, build.Env); err != nil {
		return errors.Wrapf(err, "failed to build for %s", target)
	}

	// Copy binary
	log.Debugf("FROM: Open %s ", "./target/"+target+"/release/"+binaryName)
	from, err := os.Open("./target/" + target + "/release/" + binaryName)
	if err != nil {
		panic(err)
	}
	defer from.Close()

	// Create dir
	binaryPath := filepath.Join(ctx.Config.Dist, target)
	_, err = os.Stat(binaryPath)
	if os.IsNotExist(err) {
		log.Debugf("./%s doesn't exist, creating empty folder", binaryPath)
		err := os.MkdirAll(binaryPath, 0755)
		if err != nil {
			return err
		}
	}

	log.Debugf("TO: Open %s ", binary)
	to, err := os.OpenFile(binary, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	defer to.Close()

	written, err := io.Copy(to, from)
	if err != nil {
		panic(err)
	}
	log.Debugf("Written: %v ", written)

	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.Binary,
		Path: binary,
		Name: binaryName,
		// TODO This is a hack right now
		Goos:   target,
		Goarch: "goarch",
		Goarm:  "",
		Extra: map[string]string{
			"Binary": build.Binary,
			"Ext":    ext,
		},
	})
	return nil
}

func run(ctx *context.Context, target string, command, env []string) error {
	/* #nosec */
	var cmd = exec.CommandContext(ctx, command[0], command[1:]...)
	var log = log.WithField("target", target).
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
