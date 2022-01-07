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

// Environment variables to pass through to exec
var passthroughEnvVars = []string{"HOME", "USER", "USERPROFILE", "TMPDIR", "TMP", "TEMP", "PATH"}

// Pipe that catalogs common artifacts as an SBOM.
type Pipe struct{}

func (Pipe) String() string { return "cataloging artifacts" }
func (Pipe) Skip(ctx *context.Context) bool {
	return ctx.SkipSBOMCataloging || len(ctx.Config.SBOMs) == 0
}

// Default sets the Pipes defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("sboms")
	for i := range ctx.Config.SBOMs {
		cfg := &ctx.Config.SBOMs[i]
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
				cfg.Args = []string{"$artifact", "--file", "$document", "--output", "spdx-json"}
			}
			if len(cfg.Env) == 0 && cfg.Artifacts == "source" || cfg.Artifacts == "archive" {
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
		case "source":
			filters = append(filters, artifact.ByType(artifact.UploadableSourceArchive))
			if len(cfg.IDs) > 0 {
				log.Warn("when artifacts is `source`, `ids` has no effect. ignoring")
			}
		case "archive":
			filters = append(filters, artifact.ByType(artifact.UploadableArchive))
		case "binary":
			filters = append(filters, artifact.ByType(artifact.UploadableBinary))
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
	env := ctx.Env.Copy()
	artifactDisplayName := "(any)"
	templater := tmpl.New(ctx).WithEnv(env)

	if a != nil {
		procPath, err := subprocessDistPath(ctx.Config.Dist, a.Path)
		if err != nil {
			return nil, fmt.Errorf("cataloging artifacts failed: cannot determine artifact path for %q: %w", a.Path, err)
		}
		env["artifact"] = procPath
		env["artifactID"] = a.ID()

		templater = templater.WithArtifact(a, nil)
		artifactDisplayName = a.Path
	}

	var paths []string
	for idx, sbom := range cfg.Documents {
		input := filepath.Join(ctx.Config.Dist, expand(sbom, env))

		path, err := templater.Apply(input)
		if err != nil {
			return nil, fmt.Errorf("cataloging artifacts failed: %s: invalid template: %w", input, err)
		}

		path, err = filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("cataloging artifacts failed: unable to create artifact path %q: %w", sbom, err)
		}

		procPath, err := subprocessDistPath(ctx.Config.Dist, path)
		if err != nil {
			return nil, fmt.Errorf("cataloging artifacts failed: cannot determine document path for %q: %w", path, err)
		}

		env[fmt.Sprintf("document%d", idx)] = procPath
		if idx == 0 {
			env["document"] = procPath
		}

		paths = append(paths, procPath)
	}

	var names []string
	for _, p := range paths {
		names = append(names, filepath.Base(p))
	}

	fields := log.Fields{"cmd": cfg.Cmd, "artifact": artifactDisplayName, "sboms": strings.Join(names, ", ")}

	// nolint:prealloc
	var args []string
	for _, arg := range cfg.Args {
		renderedArg, err := templater.Apply(expand(arg, env))
		if err != nil {
			return nil, fmt.Errorf("cataloging artifacts failed: %s: invalid template: %w", arg, err)
		}
		args = append(args, renderedArg)
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
	cmd.Env = append(cmd.Env, cfg.Env...)
	cmd.Dir = ctx.Config.Dist

	var b bytes.Buffer
	w := gio.Safe(&b)
	cmd.Stderr = io.MultiWriter(logext.NewWriter(fields, logext.Error), w)
	cmd.Stdout = io.MultiWriter(logext.NewWriter(fields, logext.Info), w)

	log.WithFields(fields).Info("cataloging")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cataloging artifacts: %s failed: %w: %s", cfg.Cmd, err, b.String())
	}

	var artifacts []*artifact.Artifact

	for _, sbom := range cfg.Documents {
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
