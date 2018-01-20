package rust

import (
	"fmt"
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

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	// only set defaults if there are rusts in the config file.
	if len(ctx.Config.Rust) == 0 {
		return nil
	}

	for i, rust := range ctx.Config.Rust {
		ctx.Config.Rust[i] = buildWithDefaults(ctx, rust)
	}
	return nil
}

func buildWithDefaults(ctx *context.Context, rust config.Rust) config.Rust {
	if rust.Binary == "" {
		rust.Binary = ctx.Config.Release.GitHub.Name
	}

	// One idea would be, if no rust.Target are defined
	// to add the default target for the current OS.
	// E.g. hit `rustup show`. The default host is shown.
	// For an up to date Mac is will show "Default host: x86_64-apple-darwin"
	return rust
}

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
	var ext = extFor(target)
	var binaryName = build.Binary + ext
	var binary = filepath.Join(ctx.Config.Dist, target, binaryName)
	log.WithField("binary", binary).Info("building")
	cmd := []string{"cargo", "build"}
	cmd = append(cmd, "--release", "--bin", binaryName, "--target", target)
	if err := run(ctx, target, cmd, build.Env); err != nil {
		return errors.Wrapf(err, "failed to build for %s", target)
	}

	// The folder dist/$target don't exist at this point,
	// because Rust compiles into a different folder (target).
	// We need to create the right folder.
	binaryPath := filepath.Join(ctx.Config.Dist, target)
	_, err := os.Stat(binaryPath)
	if os.IsNotExist(err) {
		log.Debugf("./%s doesn't exist, creating empty folder", binaryPath)
		err := os.MkdirAll(binaryPath, 0755)
		if err != nil {
			return err
		}
	}

	// When you compile a rust app with cargo
	// the result is stored in a `target` folder.
	// To enable this tool to build archives,
	// we copy the binary into the dist/-folder.
	err = copyBinary(target, binary, binaryName)
	if err != nil {
		return nil
	}

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

func copyBinary(target, binary, binaryName string) error {
	source := fmt.Sprintf("./target/%s/release/%s", target, binaryName)
	log.Debugf("Copy binary from %s to %s for target %s", source, binary, target)

	from, err := os.Open(source)
	if err != nil {
		return err
	}
	defer from.Close()

	// 0755 because the `cargo build` command creates
	// binaries with these permissions
	to, err := os.OpenFile(binary, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		return err
	}
	return nil
}

// For returns the binary extension for the given target
// Right now this is a modified version of goreleaser/internal/ext
func extFor(target string) string {
	if strings.Contains(target, "windows") {
		return ".exe"
	}
	return ""
}
