package sign

import (
	"fmt"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/git"
	"github.com/goreleaser/goreleaser/v2/internal/ids"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const defaultSignatureName = `${artifact}_{{ .Os }}_{{ .Arch }}{{ with .Arm }}v{{ . }}{{ end }}{{ with .Mips }}_{{ . }}{{ end }}{{ if not (eq .Amd64 "v1") }}{{ .Amd64 }}{{ end }}`

// BinaryPipe that signs binary images and manifests.
type BinaryPipe struct{}

func (BinaryPipe) String() string { return "signing binaries" }

func (BinaryPipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Sign) || len(ctx.Config.BinarySigns) == 0
}

func (BinaryPipe) Dependencies(ctx *context.Context) []string {
	var cmds []string
	for _, s := range ctx.Config.BinarySigns {
		cmds = append(cmds, s.Cmd)
	}
	return cmds
}

// Default sets the Pipes defaults.
func (BinaryPipe) Default(ctx *context.Context) error {
	gpgPath, _ := git.Clean(git.Run(ctx, "config", "gpg.program"))
	if gpgPath == "" {
		gpgPath = defaultGpg
	}

	ids := ids.New("binary_signs")
	for i := range ctx.Config.BinarySigns {
		cfg := &ctx.Config.BinarySigns[i]
		if cfg.Cmd == "" {
			// gpgPath is either "gpg" (default) or the user's git config gpg.program value
			cfg.Cmd = gpgPath
		}
		if cfg.Signature == "" {
			cfg.Signature = defaultSignatureName
		}
		if len(cfg.Args) == 0 {
			cfg.Args = []string{"--output", "$signature", "--detach-sig", "$artifact"}
		}
		if cfg.Artifacts == "" {
			cfg.Artifacts = "binary"
		}
		if cfg.ID == "" {
			cfg.ID = "default"
		}
		ids.Inc(cfg.ID)
	}
	return ids.Validate()
}

// Run signs and pushes the binary images signatures.
func (BinaryPipe) Run(ctx *context.Context) error {
	g := semerrgroup.New(ctx.Parallelism)
	for i := range ctx.Config.BinarySigns {
		cfg := ctx.Config.BinarySigns[i]
		g.Go(func() error {
			switch cfg.Artifacts {
			case "binary":
				// do nothing
			case "none":
				return pipe.ErrSkipSignEnabled
			default:
				return fmt.Errorf("invalid list of artifacts to sign: %s", cfg.Artifacts)
			}
			filters := []artifact.Filter{artifact.ByType(artifact.Binary)}
			if len(cfg.IDs) > 0 {
				filters = append(filters, artifact.ByIDs(cfg.IDs...))
			}
			return sign(ctx, config.Sign(cfg), ctx.Artifacts.Filter(artifact.And(filters...)).List())
		})
	}
	return g.Wait()
}
