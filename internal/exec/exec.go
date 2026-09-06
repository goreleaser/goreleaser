// Package exec can execute commands on the OS.
package exec

import (
	"bytes"
	"io"
	"os"
	"os/exec"

	"github.com/caarlos0/log"
	"github.com/goreleaser/go-shellwords"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/extrafiles"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/internal/gio"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/redact"
	"github.com/goreleaser/goreleaser/v2/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/v2/internal/summary"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Environment variables to pass through to exec
var passthroughEnvVars = []string{"HOME", "USER", "USERPROFILE", "TMPDIR", "TMP", "TEMP", "PATH", "SYSTEMROOT"}

// Execute the given publisher
func Execute(ctx *context.Context, publishers []config.Publisher) error {
	skips := pipe.SkipMemento{}
	for _, p := range publishers {
		log.WithField("name", p.Name).Debug("executing custom publisher")
		err := executePublisher(ctx, p)
		if err != nil && pipe.IsSkip(err) {
			skips.Remember(err)
			continue
		}
		if err != nil {
			return err
		}
	}
	return skips.Evaluate()
}

func executePublisher(ctx *context.Context, publisher config.Publisher) error {
	disabled, err := tmpl.New(ctx).Bool(publisher.Disable)
	if err != nil {
		return err
	}
	if disabled {
		return pipe.Skip("publisher is disabled")
	}

	log.Debugf("filtering %d artifacts", len(ctx.Artifacts.List()))
	artifacts := filterArtifacts(ctx, publisher)

	extraFiles, err := extrafiles.Find(ctx, publisher.ExtraFiles)
	if err != nil {
		return err
	}

	for name, path := range extraFiles {
		artifacts = append(artifacts, &artifact.Artifact{
			Name: name,
			Path: path,
			Type: artifact.UploadableFile,
		})
	}

	log.Debugf("will execute custom publisher with %d artifacts", len(artifacts))

	g := semerrgroup.New(ctx.Parallelism)
	for _, artifact := range artifacts {
		g.Go(func() error {
			c, err := resolveCommand(ctx, publisher, artifact)
			if err != nil {
				return err
			}

			return executeCommand(c, artifact)
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}
	if len(artifacts) > 0 {
		summary.Appendf("Ran custom publisher `%s` on %d artifacts", publisher.Name, len(artifacts))
	}
	return nil
}

func executeCommand(c *command, artifact *artifact.Artifact) error {
	//nolint:gosec
	cmd := exec.CommandContext(c.Ctx, c.Args[0], c.Args[1:]...)
	cmd.Env = []string{}
	for _, key := range passthroughEnvVars {
		if value := os.Getenv(key); value != "" {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	cmd.Env = append(cmd.Env, c.Env...)

	if c.Dir != "" {
		cmd.Dir = c.Dir
	}

	log.WithField("args", redactArgs(c.Args, cmd.Env)).
		WithField("artifact", artifact.Name).
		Debug("executing command")

	var b bytes.Buffer
	w := gio.Safe(&b)
	stderr := redact.Writer(io.MultiWriter(logext.NewWriter(), w), cmd.Env)
	stdout := redact.Writer(io.MultiWriter(logext.NewWriter(), w), cmd.Env)
	cmd.Stderr = stderr
	cmd.Stdout = stdout

	log := log.WithField("cmd", c.Args[0]).
		WithField("artifact", artifact.Name)

	log.Info("publishing")
	runErr := cmd.Run()
	stderrErr := stderr.Close()
	stdoutErr := stdout.Close()
	if runErr != nil {
		return gerrors.Wrap(
			runErr,
			gerrors.WithMessage("publishing failed"),
			gerrors.WithDetails("cmd", cmd.Args[0]),
			gerrors.WithOutput(b.String()),
		)
	}
	if stderrErr != nil {
		return stderrErr
	}
	if stdoutErr != nil {
		return stdoutErr
	}

	log.Debug("command finished successfully")
	return nil
}

func redactArgs(args, env []string) []string {
	redacted := make([]string, len(args))
	for i, arg := range args {
		redacted[i] = redact.String(arg, env)
	}
	return redacted
}

func filterArtifacts(ctx *context.Context, publisher config.Publisher) []*artifact.Artifact {
	types := []artifact.Type{
		artifact.UploadableArchive,
		artifact.UploadableFile,
		artifact.LinuxPackage,
		artifact.UploadableBinary,
		artifact.DockerImage,
		artifact.DockerManifest,
		artifact.DockerImageV2,
		artifact.UploadableSourceArchive,
		artifact.SBOM,
		artifact.PySdist,
		artifact.PyWheel,
	}

	if publisher.Checksum {
		types = append(types, artifact.Checksum)
	}

	if publisher.Meta {
		types = append(types, artifact.Metadata)
	}

	if publisher.Signature {
		types = append(types, artifact.Signature, artifact.Certificate)
	}

	return ctx.Artifacts.Filter(artifact.And(
		artifact.ByTypes(types...),
		artifact.ByIDs(publisher.IDs...),
	)).List()
}

type command struct {
	Ctx  *context.Context
	Dir  string
	Env  []string
	Args []string
}

// resolveCommand returns the a command based on publisher template with replaced variables
// Those variables can be replaced by the given context, goos, goarch, goarm and more.
func resolveCommand(ctx *context.Context, publisher config.Publisher, artifact *artifact.Artifact) (*command, error) {
	var err error
	dir := publisher.Dir

	tpl := tmpl.New(ctx).WithArtifact(artifact)
	if dir != "" {
		dir, err = tpl.Apply(dir)
		if err != nil {
			return nil, err
		}
	}

	cmd := publisher.Cmd
	if cmd != "" {
		cmd, err = tpl.Apply(cmd)
		if err != nil {
			return nil, err
		}
	}

	args, err := shellwords.Parse(cmd)
	if err != nil {
		return nil, err
	}

	env := make([]string, len(publisher.Env))
	for i, e := range publisher.Env {
		e, err = tpl.Apply(e)
		if err != nil {
			return nil, err
		}
		env[i] = e
	}

	return &command{
		Ctx:  ctx,
		Dir:  dir,
		Env:  env,
		Args: args,
	}, nil
}
