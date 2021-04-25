package sign

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe for artifact signing.
type Pipe struct{}

func (Pipe) String() string {
	return "signing artifacts"
}

// Default sets the Pipes defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("signs")
	for i := range ctx.Config.Signs {
		cfg := &ctx.Config.Signs[i]
		if cfg.Cmd == "" {
			cfg.Cmd = "gpg"
		}
		if cfg.Signature == "" {
			cfg.Signature = "${artifact}.sig"
		}
		if len(cfg.Args) == 0 {
			cfg.Args = []string{"--output", "$signature", "--detach-sig", "$artifact"}
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
	if ctx.SkipSign {
		return pipe.ErrSkipSignEnabled
	}

	g := semerrgroup.New(ctx.Parallelism)
	for i := range ctx.Config.Signs {
		cfg := ctx.Config.Signs[i]
		g.Go(func() error {
			var filters []artifact.Filter
			switch cfg.Artifacts {
			case "checksum":
				filters = append(filters, artifact.ByType(artifact.Checksum))
				if len(cfg.IDs) > 0 {
					log.Warn("when artifacts is `checksum`, `ids` has no effect. ignoring")
				}
			case "source":
				filters = append(filters, artifact.ByType(artifact.UploadableSourceArchive))
				if len(cfg.IDs) > 0 {
					log.Warn("when artifacts is `source`, `ids` has no effect. ignoring")
				}
			case "all":
				filters = append(filters, artifact.Or(
					artifact.ByType(artifact.UploadableArchive),
					artifact.ByType(artifact.UploadableBinary),
					artifact.ByType(artifact.UploadableSourceArchive),
					artifact.ByType(artifact.Checksum),
					artifact.ByType(artifact.LinuxPackage),
				))
				if len(cfg.IDs) > 0 {
					filters = append(filters, artifact.ByIDs(cfg.IDs...))
				}
			case "none":
				return pipe.ErrSkipSignEnabled
			default:
				return fmt.Errorf("invalid list of artifacts to sign: %s", cfg.Artifacts)
			}
			return sign(ctx, cfg, ctx.Artifacts.Filter(artifact.And(filters...)).List())
		})
	}
	return g.Wait()
}

func sign(ctx *context.Context, cfg config.Sign, artifacts []*artifact.Artifact) error {
	for _, a := range artifacts {
		artifact, err := signone(ctx, cfg, a)
		if err != nil {
			return err
		}
		ctx.Artifacts.Add(artifact)
	}
	return nil
}

func signone(ctx *context.Context, cfg config.Sign, a *artifact.Artifact) (*artifact.Artifact, error) {
	env := ctx.Env.Copy()
	env["artifact"] = a.Path
	env["signature"] = expand(cfg.Signature, env)

	// nolint:prealloc
	var args []string
	for _, a := range cfg.Args {
		arg := expand(a, env)
		arg, err := tmpl.New(ctx).WithEnv(env).Apply(arg)
		if err != nil {
			return nil, fmt.Errorf("sign failed: %s: invalid template: %w", a, err)
		}
		args = append(args, arg)
	}

	var stdin io.Reader
	if cfg.Stdin != nil {
		stdin = strings.NewReader(*cfg.Stdin)
	} else if cfg.StdinFile != "" {
		f, err := os.Open(cfg.StdinFile)
		if err != nil {
			return nil, fmt.Errorf("sign failed: cannot open file %s: %w", cfg.StdinFile, err)
		}
		defer f.Close()

		stdin = f
	}

	// The GoASTScanner flags this as a security risk.
	// However, this works as intended. The nosec annotation
	// tells the scanner to ignore this.
	// #nosec
	cmd := exec.CommandContext(ctx, cfg.Cmd, args...)
	cmd.Stderr = logext.NewWriter(log.WithField("cmd", cfg.Cmd))
	cmd.Stdout = cmd.Stderr
	if stdin != nil {
		cmd.Stdin = stdin
	}
	log.WithField("cmd", cmd.Args).Info("signing")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("sign: %s failed", cfg.Cmd)
	}

	artifactPathBase, _ := filepath.Split(a.Path)

	env["artifact"] = a.Name
	name := expand(cfg.Signature, env)

	sigFilename := filepath.Base(env["signature"])
	return &artifact.Artifact{
		Type: artifact.Signature,
		Name: name,
		Path: filepath.Join(artifactPathBase, sigFilename),
		Extra: map[string]interface{}{
			"ID": cfg.ID,
		},
	}, nil
}

func expand(s string, env map[string]string) string {
	return os.Expand(s, func(key string) string {
		return env[key]
	})
}
