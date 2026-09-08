package docker

import (
	stdctx "context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"

	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Template fields exposed for base image annotations
// (e.g., org.opencontainers.image.base.{name,digest}).
const (
	keyBaseImage       = "BaseImage"
	keyBaseImageDigest = "BaseImageDigest"
	// Template fields exposed to docker hooks.
	keyImages     = "Images"
	keyDockerfile = "Dockerfile"
	keyDigest     = "Digest"
	keyContextDir = "ContextDir"
)

// errNoBaseImage is returned when the Dockerfile has no resolvable base image
// (scratch, no FROM, parse miss). Callers can silence this with errors.Is.
var errNoBaseImage = errors.New("no base image")

type dockerImage struct{ name, digest string }

// getBaseImage returns the base image of dockerfile and its manifest digest.
// Returns errNoBaseImage when there's no usable FROM. Returns (base, "", err)
// on digest resolution failure, so callers can still use the image name.
func getBaseImage(ctx *context.Context, dockerfile string, buildArgs map[string]string) (dockerImage, error) {
	content, err := os.ReadFile(dockerfile)
	if err != nil {
		return dockerImage{}, err
	}
	overrides, err := baseImageBuildArgs(ctx, string(content), buildArgs)
	if err != nil {
		return dockerImage{}, err
	}
	base := parseBaseImage(string(content), overrides)
	if base == "" || strings.EqualFold(base, "scratch") {
		return dockerImage{}, errNoBaseImage
	}
	if _, digest, ok := strings.Cut(base, "@"); ok && strings.HasPrefix(digest, "sha256:") {
		return dockerImage{base, digest}, nil
	}
	digest, err := resolveBaseImageDigest(ctx, base)
	if err != nil {
		return dockerImage{name: base}, err
	}
	return dockerImage{base, digest}, nil
}

// Other build arguments can depend on the resolved base image and are rendered later.
func baseImageBuildArgs(ctx *context.Context, content string, args map[string]string) (map[string]string, error) {
	used := map[string]bool{}
	for line := range strings.SplitSeq(continuationRe.ReplaceAllString(content, " "), "\n") {
		if m := fromRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			os.Expand(m[1], func(name string) string {
				key, _, _ := strings.Cut(name, ":-")
				used[key] = true
				return ""
			})
		}
	}
	if len(used) == 0 {
		return nil, nil
	}

	tpl := tmpl.New(ctx)
	result := map[string]string{}
	for _, key := range slices.Sorted(maps.Keys(args)) {
		k, err := tpl.Apply(key)
		if err != nil {
			return nil, err
		}
		if !used[k] {
			continue
		}
		v, err := tpl.Apply(args[key])
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(v) != "" {
			result[k] = v
		}
	}
	return result, nil
}

var (
	continuationRe = regexp.MustCompile(`\\\s*\n`)
	argRe          = regexp.MustCompile(`(?i)^ARG\s+([A-Za-z_][A-Za-z0-9_]*)(?:=(.*))?$`)
	fromRe         = regexp.MustCompile(`(?i)^FROM(?:\s+--\S+)*\s+(\S+)(?:\s+AS\s+(\S+))?\s*$`)
)

// parseBaseImage returns the final stage's base image, following AS aliases and
// substituting effective global ARG values. Returns "" if no FROM is found.
// Doesn't try to be a full Dockerfile parser — only enough to fill the
// BaseImage/BaseImageDigest template vars. The real `docker build` is the
// source of truth.
func parseBaseImage(content string, overrides map[string]string) string {
	content = continuationRe.ReplaceAllString(content, " ")

	args := map[string]string{}
	aliases := map[string]string{}
	var base string

	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if m := argRe.FindStringSubmatch(line); m != nil && base == "" {
			// Only global ARGs (before any FROM) are usable in FROM lines.
			args[m[1]] = strings.Trim(m[2], `"'`)
			if override := overrides[m[1]]; override != "" {
				args[m[1]] = override
			}
			continue
		}

		if m := fromRe.FindStringSubmatch(line); m != nil {
			base = substituteArgs(m[1], args)
			if alias := m[2]; alias != "" {
				aliases[strings.ToLower(alias)] = base
			}
		}
	}

	for range len(aliases) + 1 {
		next, ok := aliases[strings.ToLower(base)]
		if !ok || next == base {
			break
		}
		base = next
	}
	return base
}

func substituteArgs(s string, args map[string]string) string {
	return os.Expand(s, func(name string) string {
		key, def, _ := strings.Cut(name, ":-")
		if v := args[key]; v != "" {
			return v
		}
		return def
	})
}

// resolveBaseImageDigest queries `docker buildx imagetools inspect` for
// the manifest digest of the given image reference.
func resolveBaseImageDigest(ctx stdctx.Context, ref string) (string, error) {
	cmd := exec.CommandContext(
		ctx,
		"docker", "buildx", "imagetools",
		"inspect", ref,
		"--format", "{{.Manifest.Digest}}",
	)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker buildx imagetools inspect %s: %w", ref, err)
	}
	digest := strings.TrimSpace(string(out))
	if !strings.HasPrefix(digest, "sha256:") {
		return "", fmt.Errorf("unexpected digest output for %s: %q", ref, digest)
	}
	return digest, nil
}
