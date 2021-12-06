package sign

import (
	"fmt"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe that signs docker images and manifests.
type DockerPipe struct{}

func (DockerPipe) String() string { return "signing docker images" }

func (DockerPipe) Skip(ctx *context.Context) bool {
	return ctx.SkipSign || len(ctx.Config.DockerSigns) == 0
}

// Default sets the Pipes defaults.
func (DockerPipe) Default(ctx *context.Context) error {
	ids := ids.New("docker_signs")
	for i := range ctx.Config.DockerSigns {
		cfg := &ctx.Config.DockerSigns[i]
		if cfg.Cmd == "" {
			cfg.Cmd = "cosign"
		}
		if len(cfg.Args) == 0 {
			cfg.Args = []string{"sign", "--key=cosign.key", "$artifact"}
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

// Publish signs and pushes the docker images signatures.
func (DockerPipe) Publish(ctx *context.Context) error {
	g := semerrgroup.New(ctx.Parallelism)
	for i := range ctx.Config.DockerSigns {
		cfg := ctx.Config.DockerSigns[i]
		g.Go(func() error {
			var filters []artifact.Filter
			switch cfg.Artifacts {
			case "images":
				filters = append(filters, artifact.ByType(artifact.DockerImage))
			case "manifests":
				filters = append(filters, artifact.ByType(artifact.DockerManifest))
			case "all":
				filters = append(filters, artifact.Or(
					artifact.ByType(artifact.DockerImage),
					artifact.ByType(artifact.DockerManifest),
				))
			case "none": // TODO(caarlos0): remove this
				return pipe.ErrSkipSignEnabled
			default:
				return fmt.Errorf("invalid list of artifacts to sign: %s", cfg.Artifacts)
			}

			if len(cfg.IDs) > 0 {
				filters = append(filters, artifact.ByIDs(cfg.IDs...))
			}
			return sign(ctx, cfg, ctx.Artifacts.Filter(artifact.And(filters...)).List())
		})
	}
	return g.Wait()
}
