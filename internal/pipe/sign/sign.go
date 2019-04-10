package sign

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe for artifact signing.
type Pipe struct{}

func (Pipe) String() string {
	return "signing artifacts"
}

// Default sets the Pipes defaults.
func (Pipe) Default(ctx *context.Context) error {
	cfg := &ctx.Config.Sign
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
	return nil
}

// Run executes the Pipe.
func (Pipe) Run(ctx *context.Context) error {
	if ctx.SkipSign {
		return pipe.ErrSkipSignEnabled
	}

	switch ctx.Config.Sign.Artifacts {
	case "checksum":
		return sign(ctx, ctx.Artifacts.Filter(artifact.ByType(artifact.Checksum)).List())
	case "all":
		return sign(ctx, ctx.Artifacts.Filter(
			artifact.Or(
				artifact.ByType(artifact.UploadableArchive),
				artifact.ByType(artifact.UploadableBinary),
				artifact.ByType(artifact.Checksum),
				artifact.ByType(artifact.LinuxPackage),
			)).List())
	case "none":
		return pipe.ErrSkipSignEnabled
	default:
		return fmt.Errorf("invalid list of artifacts to sign: %s", ctx.Config.Sign.Artifacts)
	}
}

func sign(ctx *context.Context, artifacts []artifact.Artifact) error {

	for _, a := range artifacts {
		artifact, err := signone(ctx, a)
		if err != nil {
			return err
		}

		ctx.Artifacts.Add(*artifact)
	}
	return nil
}

func signone(ctx *context.Context, a artifact.Artifact) (*artifact.Artifact, error) {
	cfg := ctx.Config.Sign

	env := map[string]string{
		"artifact": a.Path,
	}
	env["signature"] = expand(cfg.Signature, env)

	// nolint:prealloc
	var args []string
	for _, a := range cfg.Args {
		args = append(args, expand(a, env))
	}

	// The GoASTScanner flags this as a security risk.
	// However, this works as intended. The nosec annotation
	// tells the scanner to ignore this.
	// #nosec
	cmd := exec.CommandContext(ctx, cfg.Cmd, args...)
	log.WithField("cmd", cmd.Args).Debug("running")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("sign: %s failed with %q", cfg.Cmd, string(output))
	}

	artifactPathBase, _ := filepath.Split(a.Path)

	env["artifact"] = a.Name
	name := expand(cfg.Signature, env)

	sigFilename := filepath.Base(env["signature"])
	return &artifact.Artifact{
		Type: artifact.Signature,
		Name: name,
		Path: filepath.Join(artifactPathBase, sigFilename),
	}, nil
}

func expand(s string, env map[string]string) string {
	return os.Expand(s, func(key string) string {
		return env[key]
	})
}
