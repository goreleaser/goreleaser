package docker

import (
	stdctx "context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Template fields exposed for base image annotations
// (e.g., org.opencontainers.image.base.{name,digest}).
const (
	keyBaseImage       = "BaseImage"
	keyBaseImageDigest = "BaseImageDigest"
)

// errNoBaseImage is returned when the Dockerfile has no resolvable base image
// (scratch, no FROM, parse miss). Callers can silence this with errors.Is.
var errNoBaseImage = errors.New("no base image")

type dockerImage struct{ name, digest string }

// getBaseImage returns the base image of dockerfile and its manifest digest.
// Returns errNoBaseImage when there's no usable FROM. Returns (base, "", err)
// on digest resolution failure, so callers can still use the image name.
func getBaseImage(ctx *context.Context, dockerfile string) (dockerImage, error) {
	content, err := os.ReadFile(dockerfile)
	if err != nil {
		return dockerImage{}, err
	}
	base := parseBaseImage(string(content))
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

var (
	continuationRe = regexp.MustCompile(`\\\s*\n`)
	argRe          = regexp.MustCompile(`(?i)^ARG\s+([A-Za-z_][A-Za-z0-9_]*)(?:=(.*))?$`)
	fromRe         = regexp.MustCompile(`(?i)^FROM(?:\s+--\S+)*\s+(\S+)(?:\s+AS\s+(\S+))?\s*$`)
)

// parseBaseImage returns the final stage's base image, following AS
// aliases and substituting global ARG defaults. Returns "" if no FROM
// is found. Doesn't try to be a full Dockerfile parser — only enough
// to fill the BaseImage/BaseImageDigest template vars. If it gets it
// wrong the user just won't get the annotation; the real `docker
// build` is the source of truth.
func parseBaseImage(content string) string {
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
