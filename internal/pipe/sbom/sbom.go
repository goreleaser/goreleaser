package sbom

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Environment variables to pass through to exec
var passthroughEnvVars = []string{"HOME", "USER", "USERPROFILE", "TMPDIR", "TMP", "TEMP", "PATH", "LOCALAPPDATA"}

// Pipe that catalogs common artifacts as an SBOM.
type Pipe struct{}

func (Pipe) String() string { return "cataloging artifacts" }
func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.SBOM) || len(ctx.Config.SBOMs) == 0
}

func (Pipe) Dependencies(ctx *context.Context) []string {
	var cmds []string
	for _, s := range ctx.Config.SBOMs {
		cmds = append(cmds, s.Cmd)
	}
	return cmds
}

// Default sets the Pipes defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("sboms")
	for i := range ctx.Config.SBOMs {
		cfg := &ctx.Config.SBOMs[i]
		if err := setConfigDefaults(cfg); err != nil {
			return err
		}
		ids.Inc(cfg.ID)
	}
	return ids.Validate()
}

func setConfigDefaults(cfg *config.SBOM) error {
	if cfg.Cmd == "" {
		cfg.Cmd = "syft"
	}
	if cfg.Artifacts == "" {
		cfg.Artifacts = "archive"
	}
	if len(cfg.Documents) == 0 {
		switch cfg.Artifacts {
		case "binary":
			cfg.Documents = []string{"{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.sbom"}
		case "any":
			cfg.Documents = []string{}
		default:
			cfg.Documents = []string{"{{ .ArtifactName }}.sbom"}
		}
	}
	if cfg.Cmd == "syft" {
		if len(cfg.Args) == 0 {
			cfg.Args = []string{"$artifact", "--output", "spdx-json=$document"}
		}
		if len(cfg.Env) == 0 && (cfg.Artifacts == "source" || cfg.Artifacts == "archive") {
			cfg.Env = []string{
				"SYFT_FILE_METADATA_CATALOGER_ENABLED=true",
			}
		}
	}
	if cfg.ID == "" {
		cfg.ID = "default"
	}

	if cfg.Artifacts != "any" && len(cfg.Documents) > 1 {
		return fmt.Errorf("multiple SBOM outputs when artifacts=%q is unsupported", cfg.Artifacts)
	}
	return nil
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
		case "source":
			filters = append(filters, artifact.ByType(artifact.UploadableSourceArchive))
			if len(cfg.IDs) > 0 {
				log.Warn("when artifacts is `source`, `ids` has no effect. ignoring")
			}
		case "archive":
			filters = append(filters, artifact.ByType(artifact.UploadableArchive))
		case "binary":
			filters = append(filters, artifact.ByBinaryLikeArtifacts(ctx.Artifacts))
		case "package":
			filters = append(filters, artifact.ByType(artifact.LinuxPackage))
		case "any":
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
		if len(artifacts) == 0 {
			log.Warn("no artifacts matching current filters")
		}
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

func subprocessDistPath(distDir string, pathRelativeToCwd string) (string, error) {
	distDir = filepath.Clean(distDir)
	pathRelativeToCwd = filepath.Clean(pathRelativeToCwd)
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(distDir) {
		distDir, err = filepath.Abs(distDir)
		if err != nil {
			return "", err
		}
	}
	relativePath, err := filepath.Rel(cwd, distDir)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(pathRelativeToCwd, relativePath+string(filepath.Separator)), nil
}

func catalogArtifact(ctx *context.Context, cfg config.SBOM, a *artifact.Artifact) ([]*artifact.Artifact, error) {
	artifactDisplayName := "(any)"
	args, envs, paths, err := applyTemplate(ctx, cfg, a)
	if err != nil {
		return nil, fmt.Errorf("cataloging artifacts failed: %w", err)
	}

	if a != nil {
		artifactDisplayName = a.Path
	}

	var names []string
	for _, p := range paths {
		names = append(names, filepath.Base(p))
	}

	// The GoASTScanner flags this as a security risk.
	// However, this works as intended. The nosec annotation
	// tells the scanner to ignore this.
	// #nosec
	cmd := exec.CommandContext(ctx, cfg.Cmd, args...)
	cmd.Env = []string{}
	for _, key := range passthroughEnvVars {
		if value := os.Getenv(key); value != "" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}
	cmd.Env = append(cmd.Env, envs...)
	cmd.Dir = ctx.Config.Dist

	log.WithField("dir", cmd.Dir).
		WithField("cmd", cmd.Args).
		Debug("running")

	var b bytes.Buffer
	w := gio.Safe(&b)
	cmd.Stderr = io.MultiWriter(logext.NewWriter(), w)
	cmd.Stdout = io.MultiWriter(logext.NewWriter(), w)

	log.WithField("cmd", cfg.Cmd).
		WithField("artifact", artifactDisplayName).
		WithField("sbom", names).
		Info("cataloging")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cataloging artifacts: %s failed: %w: %s", cfg.Cmd, err, b.String())
	}

	var artifacts []*artifact.Artifact

	for _, path := range paths {
		if !filepath.IsAbs(path) {
			path = filepath.Join(ctx.Config.Dist, path)
		}

		matches, err := filepath.Glob(path)
		if err != nil {
			return nil, fmt.Errorf("cataloging artifacts: failed to find SBOM artifact %q: %w", path, err)
		}
		for _, match := range matches {
			artifacts = append(artifacts, &artifact.Artifact{
				Type: artifact.SBOM,
				Name: filepath.Base(path),
				Path: match,
				Extra: map[string]interface{}{
					artifact.ExtraID: cfg.ID,
				},
			})
		}

	}

	if len(artifacts) == 0 {
		return nil, fmt.Errorf("cataloging artifacts: command did not write any files, check your configuration")
	}

	return artifacts, nil
}

func applyTemplate(ctx *context.Context, cfg config.SBOM, a *artifact.Artifact) ([]string, []string, []string, error) {
	env := ctx.Env.Copy()
	var extraEnvs []string
	templater := tmpl.New(ctx).WithEnv(env)

	if a != nil {
		procPath, err := subprocessDistPath(ctx.Config.Dist, a.Path)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("cataloging artifacts failed: cannot determine artifact path for %q: %w", a.Path, err)
		}
		extraEnvs = appendExtraEnv("artifact", procPath, extraEnvs, env)
		extraEnvs = appendExtraEnv("artifactID", a.ID(), extraEnvs, env)
		templater = templater.WithArtifact(a)
	}

	for _, keyValue := range cfg.Env {
		renderedKeyValue, err := templater.Apply(expand(keyValue, env))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("env %q: invalid template: %w", keyValue, err)
		}
		extraEnvs = append(extraEnvs, renderedKeyValue)

		k, v, _ := strings.Cut(renderedKeyValue, "=")
		env[k] = v
	}

	var paths []string
	for idx, sbom := range cfg.Documents {
		input := expand(sbom, env)
		if !filepath.IsAbs(input) {
			// assume any absolute path is handled correctly and assume that any relative path is not already
			// adjusted to reference the dist path
			input = filepath.Join(ctx.Config.Dist, input)
		}

		path, err := templater.Apply(input)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("input %q: invalid template: %w", input, err)
		}

		path, err = filepath.Abs(path)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("unable to create artifact path %q: %w", sbom, err)
		}

		procPath, err := subprocessDistPath(ctx.Config.Dist, path)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("cannot determine document path for %q: %w", path, err)
		}

		extraEnvs = appendExtraEnv(fmt.Sprintf("document%d", idx), procPath, extraEnvs, env)
		if idx == 0 {
			extraEnvs = appendExtraEnv("document", procPath, extraEnvs, env)
		}

		paths = append(paths, procPath)
	}

	// nolint:prealloc
	var args []string
	for _, arg := range cfg.Args {
		renderedArg, err := templater.Apply(expand(arg, env))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("arg %q: invalid template: %w", arg, err)
		}
		args = append(args, renderedArg)
	}

	return args, extraEnvs, paths, nil
}

func appendExtraEnv(key, value string, envs []string, env map[string]string) []string {
	env[key] = value
	return append(envs, fmt.Sprintf("%s=%s", key, value))
}

func expand(s string, env map[string]string) string {
	return os.Expand(s, func(key string) string {
		return env[key]
	})
}
