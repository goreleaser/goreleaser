package docker

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/deprecate"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// ErrNoDocker is shown when docker cannot be found in $PATH.
var ErrNoDocker = errors.New("docker not present in $PATH")

// Pipe for docker.
type Pipe struct{}

func (Pipe) String() string {
	return "docker images"
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.Dockers {
		var docker = &ctx.Config.Dockers[i]

		if docker.Goos == "" {
			docker.Goos = "linux"
		}
		if docker.Goarch == "" {
			docker.Goarch = "amd64"
		}
		if docker.Dockerfile == "" {
			docker.Dockerfile = "Dockerfile"
		}
		if len(docker.Binaries) > 0 {
			deprecate.Notice(ctx, "docker.binaries")
		}
		if len(docker.Builds) > 0 {
			deprecate.Notice(ctx, "docker.builds")
			docker.IDs = append(docker.IDs, docker.Builds...)
		}
		for _, f := range docker.Files {
			if f == "." || strings.HasPrefix(f, ctx.Config.Dist) {
				return fmt.Errorf("invalid docker.files: can't be . or inside dist folder: %s", f)
			}
		}
	}
	return nil
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	if len(ctx.Config.Dockers) == 0 || len(ctx.Config.Dockers[0].ImageTemplates) == 0 {
		return pipe.Skip("docker section is not configured")
	}
	_, err := exec.LookPath("docker")
	if err != nil {
		return ErrNoDocker
	}
	return doRun(ctx)
}

// Publish the docker images.
func (Pipe) Publish(ctx *context.Context) error {
	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}
	var images = ctx.Artifacts.Filter(artifact.ByType(artifact.PublishableDockerImage)).List()
	for _, image := range images {
		if err := dockerPush(ctx, image); err != nil {
			return err
		}
	}
	return nil
}

func doRun(ctx *context.Context) error {
	var g = semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for _, docker := range ctx.Config.Dockers {
		docker := docker
		g.Go(func() error {
			log.WithField("docker", docker).Debug("looking for artifacts matching")
			var filters = []artifact.Filter{
				artifact.ByGoos(docker.Goos),
				artifact.ByGoarch(docker.Goarch),
				artifact.ByGoarm(docker.Goarm),
				artifact.Or(
					artifact.ByType(artifact.Binary),
					artifact.ByType(artifact.LinuxPackage),
				),
			}
			if len(docker.IDs) > 0 {
				filters = append(filters, artifact.ByIDs(docker.IDs...))
			}
			var artifacts = ctx.Artifacts.Filter(artifact.And(filters...))
			log.WithField("artifacts", artifacts.Paths()).Debug("found artifacts")
			return process(ctx, docker, artifacts.List())
		})
	}
	return g.Wait()
}

func process(ctx *context.Context, docker config.Docker, artifacts []*artifact.Artifact) error {
	tmp, err := ioutil.TempDir(ctx.Config.Dist, "goreleaserdocker")
	if err != nil {
		return fmt.Errorf("failed to create temporary dir: %w", err)
	}
	log.Debug("tempdir: " + tmp)

	images, err := processImageTemplates(ctx, docker)
	if err != nil {
		return err
	}

	if err := os.Link(docker.Dockerfile, filepath.Join(tmp, "Dockerfile")); err != nil {
		return fmt.Errorf("failed to link dockerfile: %w", err)
	}
	for _, file := range docker.Files {
		if err := os.MkdirAll(filepath.Join(tmp, filepath.Dir(file)), 0755); err != nil {
			return fmt.Errorf("failed to link extra file '%s': %w", file, err)
		}
		if err := link(file, filepath.Join(tmp, file)); err != nil {
			return fmt.Errorf("failed to link extra file '%s': %w", file, err)
		}
	}
	for _, art := range artifacts {
		if err := os.Link(art.Path, filepath.Join(tmp, filepath.Base(art.Path))); err != nil {
			return fmt.Errorf("failed to link artifact: %w", err)
		}
	}

	buildFlags, err := processBuildFlagTemplates(ctx, docker)
	if err != nil {
		return err
	}

	if err := dockerBuild(ctx, tmp, images, buildFlags, docker.Buildx); err != nil {
		return err
	}

	if strings.TrimSpace(docker.SkipPush) == "true" {
		return pipe.Skip("docker.skip_push is set")
	}
	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}
	if strings.TrimSpace(docker.SkipPush) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("prerelease detected with 'auto' push, skipping docker publish")
	}
	for _, img := range images {
		ctx.Artifacts.Add(&artifact.Artifact{
			Type:   artifact.PublishableDockerImage,
			Name:   img,
			Path:   img,
			Goarch: docker.Goarch,
			Goos:   docker.Goos,
			Goarm:  docker.Goarm,
		})
	}
	return nil
}

func processImageTemplates(ctx *context.Context, docker config.Docker) ([]string, error) {
	// nolint:prealloc
	var images []string
	for _, imageTemplate := range docker.ImageTemplates {
		image, err := tmpl.New(ctx).Apply(imageTemplate)
		if err != nil {
			return nil, fmt.Errorf("failed to execute image template '%s': %w", imageTemplate, err)
		}
		if image == "" {
			continue
		}

		images = append(images, image)
	}

	if len(images) == 0 {
		return images, errors.New("no image templates found")
	}

	return images, nil
}

func processBuildFlagTemplates(ctx *context.Context, docker config.Docker) ([]string, error) {
	// nolint:prealloc
	var buildFlags []string
	for _, buildFlagTemplate := range docker.BuildFlagTemplates {
		buildFlag, err := tmpl.New(ctx).Apply(buildFlagTemplate)
		if err != nil {
			return nil, fmt.Errorf("failed to process build flag template '%s': %w", buildFlagTemplate, err)
		}
		buildFlags = append(buildFlags, buildFlag)
	}
	return buildFlags, nil
}

// walks the src, recreating dirs and hard-linking files.
func link(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// We have the following:
		// - src = "a/b"
		// - dest = "dist/linuxamd64/b"
		// - path = "a/b/c.txt"
		// So we join "a/b" with "c.txt" and use it as the destination.
		var dst = filepath.Join(dest, strings.Replace(path, src, "", 1))
		log.WithFields(log.Fields{
			"src": path,
			"dst": dst,
		}).Debug("extra file")
		if info.IsDir() {
			return os.MkdirAll(dst, info.Mode())
		}
		return os.Link(path, dst)
	})
}

func dockerBuild(ctx *context.Context, root string, images, flags []string, buildx bool) error {
	log.WithField("image", images[0]).WithField("buildx", buildx).Info("building docker image")
	/* #nosec */
	var cmd = exec.CommandContext(ctx, "docker", buildCommand(buildx, images, flags)...)
	cmd.Dir = root
	log.WithField("cmd", cmd.Args).WithField("cwd", cmd.Dir).Debug("running")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to build docker image: %s: \n%s: %w", images[0], string(out), err)
	}
	log.Debugf("docker build output: \n%s", string(out))
	return nil
}

func buildCommand(buildx bool, images, flags []string) []string {
	base := []string{"build", "."}
	if buildx {
		base = []string{"buildx", "build", ".", "--load"}
	}
	for _, image := range images {
		base = append(base, "-t", image)
	}
	base = append(base, flags...)
	return base
}

func dockerPush(ctx *context.Context, image *artifact.Artifact) error {
	log.WithField("image", image.Name).Info("pushing docker image")
	/* #nosec */
	var cmd = exec.CommandContext(ctx, "docker", "push", image.Name)
	log.WithField("cmd", cmd.Args).Debug("running")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push docker image: \n%s: %w", string(out), err)
	}
	log.Debugf("docker push output: \n%s", string(out))
	ctx.Artifacts.Add(&artifact.Artifact{
		Type:   artifact.DockerImage,
		Name:   image.Name,
		Path:   image.Path,
		Goarch: image.Goarch,
		Goos:   image.Goos,
		Goarm:  image.Goarm,
	})
	return nil
}
