package sign

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
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe that signs common artifacts.
type Pipe struct{}

func (Pipe) String() string                 { return "signing artifacts" }
func (Pipe) Skip(ctx *context.Context) bool { return ctx.SkipSign || len(ctx.Config.Signs) == 0 }

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
			case "archive":
				filters = append(filters, artifact.ByType(artifact.UploadableArchive))
			case "binary":
				filters = append(filters, artifact.ByType(artifact.UploadableBinary))
			case "package":
				filters = append(filters, artifact.ByType(artifact.LinuxPackage))
			case "none": // TODO(caarlos0): this is not very useful, lets remove it.
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

func sign(ctx *context.Context, cfg config.Sign, artifacts []*artifact.Artifact) error {
	for _, a := range artifacts {
		artifact, err := signone(ctx, cfg, a)
		if err != nil {
			return err
		}
		if artifact != nil {
			ctx.Artifacts.Add(artifact)
		}
	}
	return nil
}

func signone(ctx *context.Context, cfg config.Sign, a *artifact.Artifact) (*artifact.Artifact, error) {
	env := ctx.Env.Copy()
	env["artifact"] = a.Path
	env["artifactID"] = a.ExtraOr("ID", "").(string)

	name, err := tmpl.New(ctx).WithEnv(env).Apply(expand(cfg.Signature, env))
	if err != nil {
		return nil, fmt.Errorf("sign failed: %s: invalid template: %w", a, err)
	}
	env["signature"] = name

	// nolint:prealloc
	var args []string
	for _, a := range cfg.Args {
		arg, err := tmpl.New(ctx).WithEnv(env).Apply(expand(a, env))
		if err != nil {
			return nil, fmt.Errorf("sign failed: %s: invalid template: %w", a, err)
		}
		args = append(args, arg)
	}

	var stdin io.Reader
	if cfg.Stdin != nil {
		s, err := tmpl.New(ctx).WithEnv(env).Apply(expand(*cfg.Stdin, env))
		if err != nil {
			return nil, err
		}
		stdin = strings.NewReader(s)
	} else if cfg.StdinFile != "" {
		f, err := os.Open(cfg.StdinFile)
		if err != nil {
			return nil, fmt.Errorf("sign failed: cannot open file %s: %w", cfg.StdinFile, err)
		}
		defer f.Close()

		stdin = f
	}

	fields := log.Fields{"cmd": cfg.Cmd, "artifact": a.Name}

	// The GoASTScanner flags this as a security risk.
	// However, this works as intended. The nosec annotation
	// tells the scanner to ignore this.
	// #nosec
	cmd := exec.CommandContext(ctx, cfg.Cmd, args...)
	var b bytes.Buffer
	w := gio.Safe(&b)
	cmd.Stderr = io.MultiWriter(logext.NewWriter(fields, logext.Error), w)
	cmd.Stdout = io.MultiWriter(logext.NewWriter(fields, logext.Info), w)
	if stdin != nil {
		cmd.Stdin = stdin
	}
	log.WithFields(fields).Info("signing")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("sign: %s failed: %w: %s", cfg.Cmd, err, b.String())
	}

	if cfg.Signature == "" {
		return nil, nil
	}

	env["artifact"] = a.Name
	name, err = tmpl.New(ctx).WithEnv(env).Apply(expand(cfg.Signature, env))
	if err != nil {
		return nil, fmt.Errorf("sign failed: %s: invalid template: %w", a, err)
	}

	artifactPathBase, _ := filepath.Split(a.Path)
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
