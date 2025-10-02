// Package exec can execute commands on the OS.
package exec

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/caarlos0/go-shellwords"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/extrafiles"
	"github.com/goreleaser/goreleaser/v2/internal/gio"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/semerrgroup"
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

	return g.Wait()
}

func executeCommand(c *command, artifact *artifact.Artifact) error {
	log.WithField("args", c.Args).
		WithField("artifact", artifact.Name).
		Debug("executing command")

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

	var b bytes.Buffer
	w := gio.Safe(&b)
	cmd.Stderr = io.MultiWriter(logext.NewWriter(), w)
	cmd.Stdout = io.MultiWriter(logext.NewWriter(), w)

	log := log.WithField("cmd", c.Args[0]).
		WithField("artifact", artifact.Name)

	log.Info("publishing")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("publishing: %s failed: %w: %s", c.Args[0], err, b.String())
	}

	log.Debug("command finished successfully")
	return nil
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
