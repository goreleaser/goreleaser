// Package exec can execute commands on the OS.
package exec

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/apex/log"
	"github.com/caarlos0/go-shellwords"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/extrafiles"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Environment variables to pass through to exec
var passthroughEnvVars = []string{"HOME", "USER", "USERPROFILE", "TMPDIR", "TMP", "TEMP", "PATH"}

// Execute the given publisher
func Execute(ctx *context.Context, publishers []config.Publisher) error {
	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}

	for _, p := range publishers {
		log.WithField("name", p.Name).Debug("executing custom publisher")
		err := executePublisher(ctx, p)
		if err != nil {
			return err
		}
	}

	return nil
}

func executePublisher(ctx *context.Context, publisher config.Publisher) error {
	log.Debugf("filtering %d artifacts", len(ctx.Artifacts.List()))
	artifacts := filterArtifacts(ctx.Artifacts, publisher)

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
		artifact := artifact
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
		WithField("env", c.Env).
		WithField("artifact", artifact.Name).
		Debug("executing command")

	// nolint: gosec
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

	fields := log.Fields{
		"cmd":      c.Args[0],
		"artifact": artifact.Name,
	}
	var b bytes.Buffer
	w := gio.Safe(&b)
	cmd.Stderr = io.MultiWriter(logext.NewWriter(fields, logext.Error), w)
	cmd.Stdout = io.MultiWriter(logext.NewWriter(fields, logext.Info), w)

	log.WithFields(fields).Info("publishing")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("publishing: %s failed: %w: %s", c.Args[0], err, b.String())
	}

	log.WithFields(fields).Debugf("command %s finished successfully", c.Args[0])
	return nil
}

func filterArtifacts(artifacts artifact.Artifacts, publisher config.Publisher) []*artifact.Artifact {
	filters := []artifact.Filter{
		artifact.ByType(artifact.UploadableArchive),
		artifact.ByType(artifact.UploadableFile),
		artifact.ByType(artifact.LinuxPackage),
		artifact.ByType(artifact.UploadableBinary),
		artifact.ByType(artifact.DockerImage),
		artifact.ByType(artifact.DockerManifest),
	}

	if publisher.Checksum {
		filters = append(filters, artifact.ByType(artifact.Checksum))
	}

	if publisher.Signature {
		filters = append(filters, artifact.ByType(artifact.Signature), artifact.ByType(artifact.Certificate))
	}

	filter := artifact.Or(filters...)

	if len(publisher.IDs) > 0 {
		filter = artifact.And(filter, artifact.ByIDs(publisher.IDs...))
	}

	return artifacts.Filter(filter).List()
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

	replacements := make(map[string]string)
	// TODO: Replacements should be associated only with relevant artifacts/archives
	archives := ctx.Config.Archives
	if len(archives) > 0 {
		replacements = archives[0].Replacements
	}

	dir := publisher.Dir
	if dir != "" {
		dir, err = tmpl.New(ctx).
			WithArtifact(artifact, replacements).
			Apply(dir)
		if err != nil {
			return nil, err
		}
	}

	cmd := publisher.Cmd
	if cmd != "" {
		cmd, err = tmpl.New(ctx).
			WithArtifact(artifact, replacements).
			Apply(cmd)
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
		e, err = tmpl.New(ctx).
			WithArtifact(artifact, replacements).
			Apply(e)
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
