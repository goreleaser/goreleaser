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
					artifact.ByType(artifact.SBOM),
				))
			case "archive":
				filters = append(filters, artifact.ByType(artifact.UploadableArchive))
			case "binary":
				filters = append(filters, artifact.ByType(artifact.UploadableBinary))
			case "sbom":
				filters = append(filters, artifact.ByType(artifact.SBOM))
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
	if err := g.Wait(); err != nil {
		return err
	}

	return ctx.Artifacts.
		Filter(artifact.ByType(artifact.Checksum)).
		Visit(func(a *artifact.Artifact) error {
			return a.Refresh()
		})
}

func sign(ctx *context.Context, cfg config.Sign, artifacts []*artifact.Artifact) error {
	for _, a := range artifacts {
		if err := a.Refresh(); err != nil {
			return err
		}
		artifacts, err := signone(ctx, cfg, a)
		if err != nil {
			return err
		}
		for _, artifact := range artifacts {
			ctx.Artifacts.Add(artifact)
		}
	}
	return nil
}

func relativeToDist(dist, f string) (string, error) {
	af, err := filepath.Abs(f)
	if err != nil {
		return "", err
	}
	df, err := filepath.Abs(dist)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(af, df) {
		return f, nil
	}
	return filepath.Join(dist, f), nil
}

func tmplPath(ctx *context.Context, env map[string]string, s string) (string, error) {
	result, err := tmpl.New(ctx).WithEnv(env).Apply(expand(s, env))
	if err != nil || result == "" {
		return "", err
	}
	return relativeToDist(ctx.Config.Dist, result)
}

func signone(ctx *context.Context, cfg config.Sign, art *artifact.Artifact) ([]*artifact.Artifact, error) {
	env := ctx.Env.Copy()
	env["artifactName"] = art.Name // shouldn't be used
	env["artifact"] = art.Path
	env["artifactID"] = art.ID()

	tmplEnv, err := templateEnvS(ctx, cfg.Env)
	if err != nil {
		return nil, fmt.Errorf("sign failed: %s: %w", art.Name, err)
	}

	for k, v := range context.ToEnv(tmplEnv) {
		env[k] = v
	}

	name, err := tmplPath(ctx, env, cfg.Signature)
	if err != nil {
		return nil, fmt.Errorf("sign failed: %s: %w", art.Name, err)
	}
	env["signature"] = name

	cert, err := tmplPath(ctx, env, cfg.Certificate)
	if err != nil {
		return nil, fmt.Errorf("sign failed: %s: %w", art.Name, err)
	}
	env["certificate"] = cert

	// nolint:prealloc
	var args []string
	for _, a := range cfg.Args {
		arg, err := tmpl.New(ctx).WithEnv(env).Apply(expand(a, env))
		if err != nil {
			return nil, fmt.Errorf("sign failed: %s: %w", art.Name, err)
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

	fields := log.Fields{"cmd": cfg.Cmd, "artifact": art.Name}
	if name != "" {
		fields["signature"] = name
	}
	if cert != "" {
		fields["certificate"] = cert
	}

	// The GoASTScanner flags this as a security risk.
	// However, this works as intended. The nosec annotation
	// tells the scanner to ignore this.
	// #nosec
	cmd := exec.CommandContext(ctx, cfg.Cmd, args...)
	var b bytes.Buffer
	w := gio.Safe(&b)
	cmd.Stderr = io.MultiWriter(logext.NewConditionalWriter(fields, logext.Error, cfg.Output), w)
	cmd.Stdout = io.MultiWriter(logext.NewConditionalWriter(fields, logext.Info, cfg.Output), w)
	if stdin != nil {
		cmd.Stdin = stdin
	}
	cmd.Env = env.Strings()
	log.WithFields(fields).Info("signing")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("sign: %s failed: %w: %s", cfg.Cmd, err, b.String())
	}

	var result []*artifact.Artifact

	// re-execute template results, using artifact desc as artifact so they eval to the actual needed file desc.
	env["artifact"] = art.Name
	name, _ = tmpl.New(ctx).WithEnv(env).Apply(expand(cfg.Signature, env))   // could never error as it passed the previous check
	cert, _ = tmpl.New(ctx).WithEnv(env).Apply(expand(cfg.Certificate, env)) // could never error as it passed the previous check

	if cfg.Signature != "" {
		result = append(result, &artifact.Artifact{
			Type: artifact.Signature,
			Name: name,
			Path: env["signature"],
			Extra: map[string]interface{}{
				artifact.ExtraID: cfg.ID,
			},
		})
	}

	if cert != "" {
		result = append(result, &artifact.Artifact{
			Type: artifact.Certificate,
			Name: cert,
			Path: env["certificate"],
			Extra: map[string]interface{}{
				artifact.ExtraID: cfg.ID,
			},
		})
	}

	return result, nil
}

func expand(s string, env map[string]string) string {
	return os.Expand(s, func(key string) string {
		return env[key]
	})
}

func templateEnvS(ctx *context.Context, s []string) ([]string, error) {
	var out []string
	for _, s := range s {
		ts, err := tmpl.New(ctx).WithEnvS(out).Apply(s)
		if err != nil {
			return nil, err
		}
		out = append(out, ts)
	}
	return out, nil
}
