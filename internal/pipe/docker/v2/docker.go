// Package docker provides the v2 of GoReleaser's docker pipe.
package docker

import (
	"bytes"
	"cmp"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
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

var dockerDigestPattern = regexp.MustCompile("sha256:[a-z0-9]{64}")

// Pipe v2 of dockers pipe.
type Pipe struct{}

// String implements pipeline.Piper.
func (p Pipe) String() string { return "docker images (v2)" }

// Dependencies implements healthcheck.Healthchecker.
func (Pipe) Dependencies(*context.Context) []string { return []string{"docker"} }

// Skip implements Skipper.
func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Docker) ||
		len(ctx.Config.DockersV2) == 0
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
	log.Warn("--snapshot is set, using local registry - this only attests the image build process")

	if runtime.GOOS == "windows" {
		return pipe.Skip("library/registry is not available for windows")
	}

	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for _, d := range ctx.Config.DockersV2 {
		g.Go(func() error {
			// XXX: could potentially use `--output=type=local,dest=./dist/dockers/id/` to output the file tree?
			// Not sure if useful or not...
			return buildAndPublish(ctx, d)
		})
	}
	return g.Wait()
}

// Publish implements publish.Publisher.
func (Pipe) Publish(ctx *context.Context) error {
	warnExperimental()
	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for _, d := range ctx.Config.DockersV2 {
		g.Go(func() error {
			return buildAndPublish(ctx, d, "--push")
		})
	}
	return g.Wait()
}

func buildAndPublish(ctx *context.Context, d config.DockerV2, extraArgs ...string) error {
	if len(d.Platforms) == 0 {
		return pipe.Skip("no platforms to build")
	}

	arg, images, err := makeArgs(ctx, d, extraArgs)
	if err != nil {
		return err
	}

	wd, err := makeContext(d, contextArtifacts(ctx, d))
	if err != nil {
		return err
	}
	defer os.RemoveAll(wd)

	out, err := retry.DoWithData(
		func() (string, error) {
			log.WithField("id", d.ID).
				Infof("creating %d images", len(images))
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
					"wd", wd,
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

	digest := dockerDigestPattern.FindString(out)
	if digest == "" {
		return gerrors.Wrap(
			err,
			"could not find digest in output",
			"id", d.ID,
			"image", images[0],
			"output", out,
			"wd", wd,
		)
	}

	for _, img := range images {
		log.WithField("image", img).
			WithField("id", d.ID).
			WithField("digest", digest).
			Info("created image")
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: img,
			Path: img,
			Type: artifact.DockerImageV2,
			Extra: map[string]any{
				artifact.ExtraID:     d.ID,
				artifact.ExtraDigest: digest,
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
		tags = []string{"latest"}
	}
	allImages := makeImageList(images, tags)

	labelFlags, err := tplMapFlags(tpl, "--label", d.Labels)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid labels: %w", err)
	}

	buildFlags, err := tplMapFlags(tpl, "--build-arg", d.BuildArgs)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid build args: %w", err)
	}

	arg := []string{
		"buildx",
		"build",
		"--platform", strings.Join(d.Platforms, ","),
		"--attest=type=sbom",
	}
	for _, img := range allImages {
		arg = append(arg, "-t", img)
	}
	arg = append(arg, extraArgs...)
	arg = append(arg, labelFlags...)
	arg = append(arg, buildFlags...)
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

func makeContext(d config.DockerV2, artifacts []*artifact.Artifact) (string, error) {
	if len(artifacts) == 0 {
		log.Warn("no binaries or packages found for the given platform - COPY/ADD may not work")
	}

	tmp, err := os.MkdirTemp("", "goreleaserdocker")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary dir: %w", err)
	}
	// NOTE: Caller is responsible for removing the temporary directory returned by this function.

	wd, _ := os.Getwd()
	if err := gio.Copy(d.Dockerfile, filepath.Join(tmp, "Dockerfile")); err != nil {
		return "", fmt.Errorf("failed to copy dockerfile: %w: %s: %s", err, wd, d.ID)
	}

	for _, file := range d.Files {
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
Please provide any feedback you might have at https://github.com/goreleaser/goreleaser/discussions/XYZ`).
		Warn(logext.Warning("dockers_v2 is experimental and subject to change"))
}
