package sign

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pipeline"
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
		return pipeline.Skip("artifact signing disabled")
	default:
		return fmt.Errorf("invalid list of artifacts to sign: %s", ctx.Config.Sign.Artifacts)
	}
}

func sign(ctx *context.Context, artifacts []artifact.Artifact) error {
	var sigs []string
	for _, a := range artifacts {
		sig, err := signone(ctx, a)
		if err != nil {
			return err
		}
		sigs = append(sigs, sig)
	}
	for _, sig := range sigs {
		ctx.Artifacts.Add(artifact.Artifact{
			Type: artifact.Signature,
			Name: sig,
			Path: filepath.Join(ctx.Config.Dist, sig),
		})
	}
	return nil
}

func signone(ctx *context.Context, artifact artifact.Artifact) (string, error) {
	cfg := ctx.Config.Sign

	env := map[string]string{
		"artifact": artifact.Path,
	}
	env["signature"] = expand(cfg.Signature, env)

	var args []string
	for _, a := range cfg.Args {
		args = append(args, expand(a, env))
	}

	// The GoASTScanner flags this as a security risk.
	// However, this works as intended. The nosec annotation
	// tells the scanner to ignore this.
	// #nosec
	cmd := exec.CommandContext(ctx, cfg.Cmd, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("sign: %s failed with %q", cfg.Cmd, string(output))
	}
	return filepath.Base(env["signature"]), nil
}

func expand(s string, env map[string]string) string {
	return os.Expand(s, func(key string) string {
		return env[key]
	})
}
