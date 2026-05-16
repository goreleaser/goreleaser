package docker

import (
	stdctx "context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Template fields exposed for base image annotations
// (e.g., org.opencontainers.image.base.{name,digest}).
const (
	keyBaseImage       = "BaseImage"
	keyBaseImageDigest = "BaseImageDigest"
)

// baseImageFields returns template fields for the base image of d.
//
// It always returns the BaseImage/BaseImageDigest keys (possibly empty) so
// templates referencing them won't error. The digest is taken from the
// FROM line when already pinned, otherwise resolved via
// `docker buildx imagetools inspect`.
func baseImageFields(ctx *context.Context, d config.DockerV2) (tmpl.Fields, error) {
	fields := tmpl.Fields{
		keyBaseImage:       "",
		keyBaseImageDigest: "",
	}

	dockerfile, err := tmpl.New(ctx).Apply(d.Dockerfile)
	if err != nil {
		return nil, fmt.Errorf("invalid dockerfile: %w", err)
	}
	if strings.TrimSpace(dockerfile) == "" {
		return fields, nil
	}

	content, err := os.ReadFile(dockerfile)
	if err != nil {
		log.WithField("dockerfile", dockerfile).
			WithError(err).
			Debug("could not read dockerfile to resolve base image")
		return fields, nil
	}

	base := parseBaseImage(string(content))
	if base == "" || strings.EqualFold(base, "scratch") {
		return fields, nil
	}
	fields[keyBaseImage] = base

	if _, digest, ok := strings.Cut(base, "@"); ok && strings.HasPrefix(digest, "sha256:") {
		fields[keyBaseImageDigest] = digest
		return fields, nil
	}

	digest, err := resolveBaseImageDigest(ctx, base)
	if err != nil {
		log.WithField("base", base).
			WithError(err).
			Warn("could not resolve base image digest")
		return fields, nil
	}
	fields[keyBaseImageDigest] = digest
	return fields, nil
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
	var froms []string
	seenFrom := false

	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if m := argRe.FindStringSubmatch(line); m != nil && !seenFrom {
			// Only global ARGs (before any FROM) are usable in FROM lines.
			args[m[1]] = strings.Trim(m[2], `"'`)
			continue
		}

		if m := fromRe.FindStringSubmatch(line); m != nil {
			seenFrom = true
			image := substituteArgs(m[1], args)
			froms = append(froms, image)
			if alias := m[2]; alias != "" {
				aliases[strings.ToLower(alias)] = image
			}
		}
	}

	if len(froms) == 0 {
		return ""
	}

	base := froms[len(froms)-1]
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
