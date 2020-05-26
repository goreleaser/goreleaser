package exec

import (
	"fmt"
	"os/exec"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/mattn/go-shellwords"
)

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
	log.Debugf("will execute custom publisher with %d artifacts", len(artifacts))

	var g = semerrgroup.New(ctx.Parallelism)
	for _, artifact := range artifacts {
		artifact := artifact
		g.Go(func() error {
			c, err := resolveCommand(ctx, publisher, artifact)
			if err != nil {
				return err
			}

			return executeCommand(c)
		})
	}

	return g.Wait()
}

func executeCommand(c *command) error {
	log.WithField("args", c.Args).
		WithField("env", c.Env).
		Debug("executing command")

	// nolint: gosec
	var cmd = exec.CommandContext(c.Ctx, c.Args[0], c.Args[1:]...)
	cmd.Env = c.Env
	if c.Dir != "" {
		cmd.Dir = c.Dir
	}

	entry := log.WithField("cmd", c.Args[0])
	cmd.Stderr = logext.NewErrWriter(entry)
	cmd.Stdout = logext.NewWriter(entry)

	log.WithField("cmd", cmd.Args).Info("publishing")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("publishing: %s failed: %w",
			c.Args[0], err)
	}

	log.Debugf("command %s finished successfully", c.Args[0])
	return nil
}

func filterArtifacts(artifacts artifact.Artifacts, publisher config.Publisher) []*artifact.Artifact {
	filters := []artifact.Filter{
		artifact.ByType(artifact.UploadableArchive),
		artifact.ByType(artifact.UploadableFile),
		artifact.ByType(artifact.LinuxPackage),
		artifact.ByType(artifact.UploadableBinary),
	}

	if publisher.Checksum {
		filters = append(filters, artifact.ByType(artifact.Checksum))
	}

	if publisher.Signature {
		filters = append(filters, artifact.ByType(artifact.Signature))
	}

	var filter = artifact.Or(filters...)

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
