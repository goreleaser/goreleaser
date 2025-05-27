package docker

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gio"
	"github.com/goreleaser/goreleaser/v2/internal/ids"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const (
	extraImageNames = "ImageNames"
	extraImageName  = "ImageName"
)

type Pipe struct{}

// String implements pipeline.Piper.
func (p Pipe) String() string { return "docker images (v2)" }

// Dependencies implements healthcheck.Healthchecker.
func (Pipe) Dependencies(ctx *context.Context) []string { return []string{"docker"} }

// String implements defaults.Defaulter.
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
			docker.Tags = []string{"latest"}
		}
		if len(docker.Platforms) == 0 {
			docker.Platforms = []string{"linux/amd64"}
		}
		ids.Inc(docker.ID)
	}
	return ids.Validate()
}

// Run implements pipeline.Piper.
func (p Pipe) Run(ctx *context.Context) error {
	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for _, d := range ctx.Config.DockersV2 {
		g.Go(func() error {
			return buildOne(ctx, d)
		})
	}
	return g.Wait()
}

var dockerDigestPattern = regexp.MustCompile("sha256:[a-z0-9]{64}")

// Publish implements publish.Publisher.
func (Pipe) Publish(ctx *context.Context) error {
	for id, arts := range ctx.Artifacts.
		Filter(artifact.ByType(artifact.PublishableDockerImageV2)).
		GroupByID() {
		var imgs []string
		for _, art := range arts {
			log.WithField("path", art.Path).Info("loading image")
			in, err := os.Open(art.Path)
			if err != nil {
				return fmt.Errorf("docker: could not load %s: %w", art.Path, err)
			}
			defer in.Close()
			cmd := exec.CommandContext(ctx, "docker", "load")
			cmd.Env = append(ctx.Env.Strings(), cmd.Environ()...)
			var b bytes.Buffer
			w := gio.Safe(&b)
			cmd.Stderr = io.MultiWriter(logext.NewWriter(), w)
			cmd.Stdout = io.MultiWriter(logext.NewWriter(), w)
			cmd.Stdin = in
			if err := cmd.Run(); err != nil {
				return pipe.NewDetailedError(
					err,
					"args", strings.Join(cmd.Args, " "),
					"output", b.String(),
				)
			}
			imgs = append(imgs, artifact.MustExtra[string](*art, extraImageName))
		}

		for _, img := range imgs {
			log.WithField("image", img).Info("pushing image")
			cmd := exec.CommandContext(ctx, "docker", "push", img)
			cmd.Env = append(ctx.Env.Strings(), cmd.Environ()...)
			var b bytes.Buffer
			w := gio.Safe(&b)
			cmd.Stderr = io.MultiWriter(logext.NewWriter(), w)
			cmd.Stdout = io.MultiWriter(logext.NewWriter(), w)
			if err := cmd.Run(); err != nil {
				return pipe.NewDetailedError(
					err,
					"args", strings.Join(cmd.Args, " "),
					"image", img,
					"output", b.String(),
				)
			}
			digest := dockerDigestPattern.FindString(b.String())
			if digest == "" {
				return pipe.NewDetailedError(
					errors.New("failed to find docker digest in docker push output"),
					"output", b.String(),
				)
			}
			art := &artifact.Artifact{
				Type: artifact.DockerImage,
				Name: img,
				Path: img,
				Extra: map[string]any{
					artifact.ExtraDigest: digest,
				},
			}
			if id != "" {
				art.Extra[artifact.ExtraID] = id
			}
			ctx.Artifacts.Add(art)
		}

		manifests := artifact.MustExtra[[]string](*arts[0], extraImageNames)
		for _, manifest := range manifests {
			log.WithField("images", imgs).
				WithField("manifest", manifest).
				Info("creating manifest")
			arg := []string{"buildx", "imagetools", "create", "-t", manifest}
			arg = append(arg, imgs...)
			cmd := exec.CommandContext(ctx, "docker", arg...)
			cmd.Env = append(ctx.Env.Strings(), cmd.Environ()...)
			var b bytes.Buffer
			w := gio.Safe(&b)
			cmd.Stderr = io.MultiWriter(logext.NewWriter(), w)
			cmd.Stdout = io.MultiWriter(logext.NewWriter(), w)
			if err := cmd.Run(); err != nil {
				return pipe.NewDetailedError(
					err,
					"args", strings.Join(cmd.Args, " "),
					"manifest", manifest,
					"output", b.String(),
				)
			}
			digest := dockerDigestPattern.FindString(b.String())
			if digest == "" {
				return pipe.NewDetailedError(
					errors.New("failed to find docker digest in docker push output"),
					"output", b.String(),
				)
			}
			art := &artifact.Artifact{
				Type: artifact.DockerManifest,
				Name: manifest,
				Path: manifest,
				Extra: map[string]any{
					artifact.ExtraDigest: digest,
				},
			}
			if id != "" {
				art.Extra[artifact.ExtraID] = id
			}

			ctx.Artifacts.Add(art)
		}
	}

	return nil
}

func buildOne(ctx *context.Context, d config.DockerV2) error {
	if len(d.Platforms) == 0 {
		return pipe.Skip("no platforms to build")
	}

	tpl := tmpl.New(ctx)
	if err := tpl.ApplyAll(
		&d.Dockerfile,
	); err != nil {
		return fmt.Errorf("docker: %w", err)
	}

	images, err := tpl.Slice(d.Images, tmpl.NonEmpty())
	if err != nil {
		return fmt.Errorf("docker: %w", err)
	}
	if len(images) == 0 {
		return pipe.Skip("no images")
	}
	tags, err := tpl.Slice(d.Tags, tmpl.NonEmpty())
	if err != nil {
		return fmt.Errorf("docker: %w", err)
	}
	if len(tags) == 0 {
		tags = []string{"latest"}
	}
	allImages := makeImageList(images, tags)

	var labelFlags []string
	for k, v := range d.Labels {
		if err := tpl.ApplyAll(&k, &v); err != nil {
			return fmt.Errorf("docker: %w", err)
		}
		labelFlags = append(labelFlags, "--label", k+"="+v)
	}

	wd, err := makeContext(d, contextArtifacts(ctx, d))
	if err != nil {
		return fmt.Errorf("docker: %w", err)
	}
	defer os.RemoveAll(wd)

	for _, plat := range d.Platforms {
		plats := strings.ReplaceAll(plat, "/", "_")
		name := d.ID + plats + ".tar"
		path := filepath.Join(ctx.Config.Dist, "dockerv2", name)
		imgTag := images[0] + ":" + tags[0] + "-" + plats
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("docker: %w", err)
		}
		apath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("docker: %w", err)
		}
		arg := []string{
			"buildx",
			"build",
			"--platform", plat,
			"-t", imgTag,
		}
		arg = append(arg, labelFlags...)
		arg = append(
			arg,
			"-f", d.Dockerfile,
			"--output", "type=docker,dest="+apath,
			".",
		)
		cmd := exec.CommandContext(ctx, "docker", arg...)
		cmd.Dir = wd
		cmd.Env = append(ctx.Env.Strings(), cmd.Environ()...)
		var b bytes.Buffer
		w := gio.Safe(&b)
		cmd.Stderr = io.MultiWriter(logext.NewWriter(), w)
		cmd.Stdout = io.MultiWriter(logext.NewWriter(), w)
		if err := cmd.Run(); err != nil {
			return pipe.NewDetailedError(
				err,
				"args", strings.Join(cmd.Args, " "),
				"image", imgTag,
				"output", b.String(),
				"path", path,
				"platform", plat,
				"wd", wd,
			)
		}

		p := parsePlatform(plat)
		log.WithField("image", allImages[0]).
			WithField("path", path).
			WithField("id", d.ID).
			Info("created docker image")
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   name,
			Path:   path,
			Goos:   p.os,
			Goarch: p.arch,
			Goarm:  p.arm,
			Type:   artifact.PublishableDockerImageV2,
			Extra: map[string]any{
				artifact.ExtraID: d.ID,
				extraImageName:   imgTag,
				extraImageNames:  allImages,
			},
		})
	}

	return nil
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

	if err := gio.Copy(
		d.Dockerfile,
		filepath.Join(tmp, "Dockerfile"),
	); err != nil {
		return "", fmt.Errorf("failed to copy dockerfile: %w", err)
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
	return strings.Join(parts, "/"), nil
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
		result.arm = parts[2]
	}
	return result
}
