package sign

import (
	"fmt"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/ids"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// DockerPipe that signs docker images and manifests.
type DockerPipe struct{}

func (DockerPipe) String() string { return "signing docker images" }

func (DockerPipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Sign) || len(ctx.Config.DockerSigns) == 0
}

func (DockerPipe) Dependencies(ctx *context.Context) []string {
	var cmds []string
	for _, s := range ctx.Config.DockerSigns {
		cmds = append(cmds, s.Cmd)
	}
	return cmds
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
			cfg.Args = []string{"sign", "--key=cosign.key", "${artifact}@${digest}", "--yes"}
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
				filters = append(filters, artifact.ByTypes(
					artifact.DockerImage,
					artifact.DockerImageV2,
				))
			case "manifests":
				filters = append(filters, artifact.ByTypes(
					artifact.DockerManifest,
					artifact.DockerImageV2,
				))
			case "all":
				filters = append(filters, artifact.ByTypes(
					artifact.DockerImage,
					artifact.DockerManifest,
					artifact.DockerImageV2,
				))
			case "none": // TODO(caarlos0): remove this
				return pipe.ErrSkipSignEnabled
			case "":
				filters = append(filters, artifact.ByType(artifact.DockerImageV2))
			default:
				return fmt.Errorf("invalid list of artifacts to sign: %s", cfg.Artifacts)
			}

			filters = append(filters, artifact.ByIDs(cfg.IDs...))
			return sign(ctx, cfg, ctx.Artifacts.Filter(artifact.And(filters...)).List())
		})
	}
	return g.Wait()
}
