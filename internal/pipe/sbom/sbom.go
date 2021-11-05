package sbom

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe that catalogs common artifacts as an SBOM.
type Pipe struct{}

func (Pipe) String() string                 { return "cataloging artifacts" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.SBOMs) == 0 }

// Default sets the Pipes defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("sboms")
	for i := range ctx.Config.SBOMs {
		cfg := &ctx.Config.SBOMs[i]
		if cfg.Cmd == "" {
			cfg.Cmd = "syft"
			//return errors.New("cataloging artifacts failed: no command specified")
		}
		if len(cfg.SBOMs) == 0 {
			var sbom string
			switch cfg.Artifacts {
			case "container":
				sbom = `{{ replace .ArtifactName "/" "-" }}.sbom`
			case "binary":
				sbom = "{{ .ArtifactName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.sbom"
			default:
				sbom = "{{ .ArtifactName }}.sbom"
			}
			cfg.SBOMs = []string{sbom}
		}
		if len(cfg.Args) == 0 {
			cfg.Args = []string{"--file", "{{ .Dist }}/$sbom0", "--output", "spdx-json", "$artifact"}
		}
		if cfg.Artifacts == "" {
			cfg.Artifacts = "none"
		}
		if cfg.ID == "" {
			cfg.ID = "default"
		}
		ids.Inc(cfg.ID)
	}
	return ids.Validate()
}

// Run executes the Pipe.
func (Pipe) Run(ctx *context.Context) error {
	g := semerrgroup.New(ctx.Parallelism)
	for _, cfg := range ctx.Config.SBOMs {
		g.Go(catalogTask(ctx, cfg))
	}
	return g.Wait()
}

func catalogTask(ctx *context.Context, cfg config.SBOM) func() error {
	return func() error {
		var filters []artifact.Filter
		switch cfg.Artifacts {
		case "all":
			filters = append(filters, artifact.Or(
				artifact.ByType(artifact.UploadableArchive),
				artifact.ByType(artifact.Binary),
				artifact.ByType(artifact.UploadableSourceArchive),
				artifact.ByType(artifact.Checksum),
				artifact.ByType(artifact.LinuxPackage),
				artifact.ByType(artifact.PublishableDockerImage),
				artifact.ByType(artifact.UnpublishableDockerImage),
			))
		case "source":
			filters = append(filters, artifact.ByType(artifact.UploadableSourceArchive))
			if len(cfg.IDs) > 0 {
				log.Warn("when artifacts is `source`, `ids` has no effect. ignoring")
			}
		case "archive":
			filters = append(filters, artifact.ByType(artifact.UploadableArchive))
		case "binary":
			filters = append(filters, artifact.ByType(artifact.Binary))
		case "package":
			filters = append(filters, artifact.ByType(artifact.LinuxPackage))
		case "container":
			filters = append(filters, artifact.Or(
				artifact.ByType(artifact.PublishableDockerImage),
				artifact.ByType(artifact.UnpublishableDockerImage),
			))
		case "none":
			newArtifacts, err := catalogArtifact(ctx, cfg, nil)
			if err != nil {
				return err
			}
			for _, newArtifact := range newArtifacts {
				ctx.Artifacts.Add(newArtifact)
			}
			return nil
		default:
			return fmt.Errorf("invalid list of artifacts to catalog: %s", cfg.Artifacts)
		}

		if len(cfg.IDs) > 0 {
			filters = append(filters, artifact.ByIDs(cfg.IDs...))
		}
		artifacts := ctx.Artifacts.Filter(artifact.And(filters...)).List()
		return catalog(ctx, cfg, artifacts)
	}
}

func catalog(ctx *context.Context, cfg config.SBOM, artifacts []*artifact.Artifact) error {
	for _, a := range artifacts {
		newArtifacts, err := catalogArtifact(ctx, cfg, a)
		if err != nil {
			return err
		}
		for _, newArtifact := range newArtifacts {
			ctx.Artifacts.Add(newArtifact)
		}
	}
	return nil
}

func catalogArtifact(ctx *context.Context, cfg config.SBOM, a *artifact.Artifact) ([]*artifact.Artifact, error) {
	env := ctx.Env.Copy()
	var fields log.Fields
	templater := tmpl.New(ctx).WithEnv(env)
	if a != nil {
		env["artifact"] = a.Path
		env["artifactID"] = a.ID()

		templater = templater.WithArtifact(a, nil)

		var names []string
		for idx, sbom := range cfg.SBOMs {
			name, err := templater.Apply(expand(sbom, env))
			if err != nil {
				return nil, fmt.Errorf("cataloging artifacts failed: %s: invalid template: %w", a, err)
			}

			env[fmt.Sprintf("sbom%d", idx)] = name
			names = append(names, name)
		}

		fields = log.Fields{"cmd": cfg.Cmd, "artifact": a.Path, "sboms": strings.Join(names, ", ")}
	} else {
		fields = log.Fields{"cmd": cfg.Cmd, "artifact": "(manual)", "sboms": strings.Join(cfg.SBOMs, ", ")}
	}

	// nolint:prealloc
	var args []string
	for _, arg := range cfg.Args {
		arg, err := templater.Apply(expand(arg, env))
		if err != nil {
			return nil, fmt.Errorf("cataloging artifacts failed: %s: invalid template: %w", arg, err)
		}
		args = append(args, arg)
	}

	// The GoASTScanner flags this as a security risk.
	// However, this works as intended. The nosec annotation
	// tells the scanner to ignore this.
	// #nosec
	cmd := exec.CommandContext(ctx, cfg.Cmd, args...)
	var b bytes.Buffer
	w := gio.Safe(&b)
	cmd.Stderr = io.MultiWriter(logext.NewWriter(fields, logext.Error), w)
	cmd.Stdout = io.MultiWriter(logext.NewWriter(fields, logext.Info), w)

	log.WithFields(fields).Info("cataloging")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cataloging artifacts: %s failed: %w: %s", cfg.Cmd, err, b.String())
	}

	if len(cfg.SBOMs) == 0 {
		return nil, nil
	}

	var artifacts []*artifact.Artifact

	for _, sbom := range cfg.SBOMs {
		templater = tmpl.New(ctx).WithEnv(env)
		if a != nil {
			env["artifact"] = a.Name
			templater = templater.WithArtifact(a, nil)
		}

		name, err := templater.Apply(expand(sbom, env))
		if err != nil {
			return nil, fmt.Errorf("cataloging artifacts failed: %s: invalid template: %w", a, err)
		}

		search := filepath.Join(ctx.Config.Dist, name)
		matches, err := filepath.Glob(search)
		if err != nil {
			return nil, fmt.Errorf("cataloging artifacts: failed to find SBOM artifact %q: %w", search, err)
		}
		for _, match := range matches {
			artifacts = append(artifacts, &artifact.Artifact{
				Type: artifact.SBOM,
				Name: name,
				Path: match,
				Extra: map[string]interface{}{
					artifact.ExtraID: cfg.ID,
				},
			})
		}

	}

	return artifacts, nil
}

func expand(s string, env map[string]string) string {
	return os.Expand(s, func(key string) string {
		return env[key]
	})
}
