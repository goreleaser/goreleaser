// Package docker provides the v2 of GoReleaser's docker pipe.
package docker

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/internal/gio"
	"github.com/goreleaser/goreleaser/v2/internal/ids"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Pipe v2 of dockers pipe.
type Pipe struct{}

// String implements pipeline.Piper.
func (p Pipe) String() string { return "docker images (v2)" }

// Dependencies implements healthcheck.Healthchecker.
func (Pipe) Dependencies(*context.Context) []string { return []string{"docker buildx"} }

// Skip implements Skipper.
func (Pipe) Skip(ctx *context.Context) bool {
	return len(ctx.Config.DockersV2) == 0 || skips.Any(ctx, skips.Docker)
}

// Default implements defaults.Defaulter.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("dockersv2")
	for i := range ctx.Config.DockersV2 {
		docker := &ctx.Config.DockersV2[i]
		if docker.ID == "" {
			docker.ID = ctx.Config.ProjectName
		}
		if docker.Dockerfile == "" {
			docker.Dockerfile = "Dockerfile"
		}
		if len(docker.Tags) == 0 {
			docker.Tags = []string{"{{.Tag}}"}
		}
		if len(docker.Platforms) == 0 {
			docker.Platforms = []string{"linux/amd64", "linux/arm64"}
		}
		docker.Retry.Attempts = cmp.Or(docker.Retry.Attempts, 10)
		docker.Retry.Delay = cmp.Or(docker.Retry.Delay, 10*time.Second)
		docker.Retry.MaxDelay = cmp.Or(docker.Retry.MaxDelay, 5*time.Minute)
		ids.Inc(docker.ID)
	}
	return ids.Validate()
}

// Run implements pipeline.Piper.
func (p Pipe) Run(ctx *context.Context) error {
	if !ctx.Snapshot {
		return pipe.Skip("non-snapshot build")
	}

	warnExperimental()
	log.Warn("snapshot build: will not push any images")

	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for i := range ctx.Config.DockersV2 {
		for _, plat := range ctx.Config.DockersV2[i].Platforms {
			g.Go(func() error {
				// buildx won't allow us to `--load` a manifest, so we create
				// one image per platform, adding it to the tags.
				d := ctx.Config.DockersV2[i]
				d.Platforms = []string{plat}
				// XXX: could potentially use `--output=type=local,dest=./dist/dockers/id/` to output the file tree?
				// Not sure if useful or not...
				return buildImage(ctx, d, "--load")
			})
		}
	}
	return g.Wait()
}

// Publish implements publish.Publisher.
func (Pipe) Publish(ctx *context.Context) error {
	warnExperimental()
	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for _, d := range ctx.Config.DockersV2 {
		g.Go(func() error {
			return buildImage(ctx, d, "--push", "--attest=type=sbom")
		})
	}
	return g.Wait()
}

func buildImage(ctx *context.Context, d config.DockerV2, extraArgs ...string) error {
	if len(d.Platforms) == 0 {
		return pipe.Skip("no platforms to build")
	}

	arg, images, err := makeArgs(ctx, d, extraArgs)
	if err != nil {
		return err
	}

	log := log.WithField("images", strings.Join(images, "\n")).
		WithField("id", d.ID)
	log.Debug("creating images")

	wd, err := makeContext(d, contextArtifacts(ctx, d))
	if err != nil {
		return err
	}
	defer os.RemoveAll(wd)

	out, err := retry.DoWithData(
		func() (string, error) {
			cmd := exec.CommandContext(ctx, "docker", arg...)
			cmd.Dir = wd
			cmd.Env = append(ctx.Env.Strings(), cmd.Environ()...)
			var b bytes.Buffer
			w := gio.Safe(&b)
			cmd.Stderr = io.MultiWriter(logext.NewWriter(), w)
			cmd.Stdout = io.MultiWriter(logext.NewWriter(), w)
			if err := cmd.Run(); err != nil {
				return "", gerrors.Wrap(
					err,
					"could not build and publish docker image",
					"args", strings.Join(cmd.Args, " "),
					"id", d.ID,
					"image", images[0],
					"output", b.String(),
				)
			}
			return b.String(), nil
		},
		retry.RetryIf(isRetriableManifestCreate),
		retry.Attempts(d.Retry.Attempts),
		retry.Delay(d.Retry.Delay),
		retry.MaxDelay(d.Retry.MaxDelay),
		retry.LastErrorOnly(true),
	)
	if err != nil {
		return err
	}

	digest, err := os.ReadFile(filepath.Join(wd, "id.txt"))
	if err != nil {
		return gerrors.Wrap(
			err,
			"could not find digest in output",
			"id", d.ID,
			"image", images[0],
			"output", out,
			"err", err,
		)
	}

	log.WithField("digest", string(digest)).
		Info("created images")
	for _, img := range images {
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: img,
			Path: img,
			Type: artifact.DockerImageV2,
			Extra: map[string]any{
				artifact.ExtraID:     d.ID,
				artifact.ExtraDigest: string(digest),
			},
		})

		// XXX: should we extract the SBOM and add its artifact as well?
		// https://docs.docker.com/build/metadata/attestations/sbom/#inspecting-sboms
	}

	return nil
}

func makeArgs(ctx *context.Context, d config.DockerV2, extraArgs []string) ([]string, []string, error) {
	tpl := tmpl.New(ctx)
	if err := tpl.ApplyAll(
		&d.Dockerfile,
	); err != nil {
		return nil, nil, fmt.Errorf("invalid dockerfile: %w", err)
	}
	if strings.TrimSpace(d.Dockerfile) == "" {
		return nil, nil, pipe.Skip("no dockerfile")
	}
	images, err := tpl.Slice(d.Images, tmpl.NonEmpty())
	if err != nil {
		return nil, nil, fmt.Errorf("invalid images: %w", err)
	}
	if len(images) == 0 {
		return nil, nil, pipe.Skip("no images")
	}
	tags, err := tpl.Slice(d.Tags, tmpl.NonEmpty())
	if err != nil {
		return nil, nil, fmt.Errorf("invalid tags: %w", err)
	}
	if len(tags) == 0 {
		return nil, nil, errors.New("no tags provided")
	}
	// Append the -platform bit to non-empty tags.
	if len(d.Platforms) == 1 && ctx.Snapshot {
		plat := strings.TrimPrefix(d.Platforms[0], "linux/")
		plat = strings.ReplaceAll(plat, "/", "")
		for j := range tags {
			tags[j] += "-" + plat
		}
	}
	allImages := makeImageList(images, tags)

	labelFlags, err := tplMapFlags(tpl, "--label", d.Labels)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid labels: %w", err)
	}

	annotationFlags, err := tplMapFlags(tpl, "--annotation", d.Annotations)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid annotations: %w", err)
	}

	buildFlags, err := tplMapFlags(tpl, "--build-arg", d.BuildArgs)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid build args: %w", err)
	}

	flags, err := tpl.Slice(d.Flags, tmpl.NonEmpty())
	if err != nil {
		return nil, nil, fmt.Errorf("invalid flags: %w", err)
	}

	arg := []string{
		"buildx",
		"build",
		"--platform", strings.Join(d.Platforms, ","),
	}
	for _, img := range allImages {
		arg = append(arg, "-t", img)
	}
	arg = append(arg, extraArgs...)
	arg = append(arg, "--iidfile=id.txt")
	arg = append(arg, labelFlags...)
	arg = append(arg, annotationFlags...)
	arg = append(arg, buildFlags...)
	arg = append(arg, flags...)
	arg = append(arg, ".")
	return arg, allImages, nil
}

func makeImageList(imgs, tags []string) []string {
	result := map[string]struct{}{}
	for _, i := range imgs {
		for _, t := range tags {
			result[i+":"+t] = struct{}{}
		}
	}
	keys := slices.Collect(maps.Keys(result))
	slices.Sort(keys)
	return keys
}

// makeContext creates a new temporary directory, copies the artifacts and any
// extra files, returning its path.
//
// The caller is responsible for removing the temporary directory.
func makeContext(d config.DockerV2, artifacts []*artifact.Artifact) (string, error) {
	if len(artifacts) == 0 {
		log.Warn("no binaries or packages found for the given platform - COPY/ADD may not work")
	}

	tmp, err := os.MkdirTemp("", "goreleaserdocker")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary dir: %w", err)
	}

	if err := gio.Copy(d.Dockerfile, filepath.Join(tmp, "Dockerfile")); err != nil {
		return "", fmt.Errorf("failed to copy dockerfile: %w: %s", err, d.ID)
	}

	for _, file := range d.ExtraFiles {
		if err := os.MkdirAll(filepath.Join(tmp, filepath.Dir(file)), 0o755); err != nil {
			return "", fmt.Errorf("failed to copy extra file '%s': %w", file, err)
		}
		if err := gio.Copy(file, filepath.Join(tmp, file)); err != nil {
			return "", fmt.Errorf("failed to copy extra file '%s': %w", file, err)
		}
	}

	for _, art := range artifacts {
		plat, err := toPlatform(art)
		if err != nil {
			return "", fmt.Errorf("failed to make dir for artifact: %w", err)
		}

		target := filepath.Join(tmp, plat, art.Name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", fmt.Errorf("failed to make dir for artifact: %w", err)
		}

		if err := gio.Copy(art.Path, target); err != nil {
			return "", fmt.Errorf("failed to copy artifact: %w", err)
		}
	}

	return tmp, nil
}

func contextArtifacts(ctx *context.Context, d config.DockerV2) []*artifact.Artifact {
	var platFilters []artifact.Filter
	for _, p := range d.Platforms {
		plat := parsePlatform(p)
		filters := []artifact.Filter{
			artifact.ByGoos(plat.os),
			artifact.ByGoarch(plat.arch),
		}
		if plat.arm != "" {
			filters = append(filters, artifact.ByGoarm(plat.arm))
		}
		platFilters = append(platFilters, artifact.And(filters...))
	}

	filters := []artifact.Filter{
		artifact.Or(platFilters...),
		artifact.Or(
			artifact.ByType(artifact.Binary),
			artifact.ByType(artifact.LinuxPackage),
			artifact.ByType(artifact.CArchive),
			artifact.ByType(artifact.CShared),
		),
	}
	if len(d.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(d.IDs...))
	}

	artifacts := ctx.Artifacts.Filter(
		artifact.Or(
			artifact.And(filters...),
			artifact.ByType(artifact.PyWheel),
		),
	)

	return artifacts.List()
}

func toPlatform(a *artifact.Artifact) (string, error) {
	var parts []string
	switch a.Goos {
	case "linux", "darwin", "windows":
		parts = append(parts, a.Goos)
	default:
		return "", fmt.Errorf("unsupported OS: %q", a.Goos)
	}
	switch a.Goarch {
	case "amd64", "arm64", "386", "ppc64le", "s390x", "riscv64":
		parts = append(parts, a.Goarch)
	case "arm":
		parts = append(parts, a.Goarch)
		switch a.Goarm {
		case "6", "7":
			parts = append(parts, "v"+a.Goarm)
		default:
			return "", fmt.Errorf("unsupported arch: arm/v%q", a.Goarm)
		}
	default:
		return "", fmt.Errorf("unsupported arch: %q", a.Goarch)
	}
	return path.Join(parts...), nil
}

type platform struct {
	os, arch string
	arm      string
}

func parsePlatform(p string) platform {
	parts := strings.Split(p, "/")
	result := platform{
		os:   parts[0],
		arch: parts[1],
	}
	if len(parts) == 3 {
		result.arm = strings.TrimPrefix(parts[2], "v")
	}
	return result
}

// tplMapFlags templates all keys and values in the given map, returning a
// slice of them with the [flag] prefix.
//
// It'll also sort keys so the resulting slice is always in the same order.
// Finally, it will also skip entries with either an empty key or value.
func tplMapFlags(tpl *tmpl.Template, flag string, m map[string]string) ([]string, error) {
	var result []string
	keys := slices.Collect(maps.Keys(m))
	slices.Sort(keys)
	for _, k := range keys {
		v := m[k]
		if err := tpl.ApplyAll(&k, &v); err != nil {
			return nil, fmt.Errorf("docker: %w", err)
		}
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		result = append(result, flag, k+"="+v)
	}
	return result, nil
}

func isRetriableManifestCreate(err error) bool {
	out := gerrors.DetailsOf(err)["output"].(string)
	return strings.Contains(out, "manifest verification failed for digest")
}

func warnExperimental() {
	log.WithField("details", `Keep an eye on the release notes if you wish to rely on this for production builds.
Please provide any feedback you might have at https://github.com/orgs/goreleaser/discussions/6005`).
		Warn(logext.Warning("dockers_v2 is experimental and subject to change"))
}
