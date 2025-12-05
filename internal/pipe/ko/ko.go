// Package ko implements the pipe interface with the intent of
// building OCI compliant images with ko.
package ko

import (
	stdctx "context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/awslabs/amazon-ecr-credential-helper/ecr-login"
	"github.com/caarlos0/log"
	"github.com/chrismellard/docker-credential-acr-env/pkg/credhelper"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/github"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/ko/pkg/build"
	"github.com/google/ko/pkg/commands/options"
	"github.com/google/ko/pkg/publish"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/deprecate"
	"github.com/goreleaser/goreleaser/v2/internal/ids"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"golang.org/x/tools/go/packages"
)

const chainguardStatic = "cgr.dev/chainguard/static"

var (
	baseImages     sync.Map
	amazonKeychain authn.Keychain = authn.NewKeychainFromHelper(ecr.NewECRHelper(ecr.WithLogger(io.Discard)))
	azureKeychain  authn.Keychain = authn.NewKeychainFromHelper(credhelper.NewACRCredentialsHelper())
	keychain                      = authn.NewMultiKeychain(
		amazonKeychain,
		authn.DefaultKeychain,
		google.Keychain,
		github.Keychain,
		azureKeychain,
	)

	errNoRepositories    = errors.New("ko: missing repositories: please set either the repository field or a $KO_DOCKER_REPO environment variable")
	errInvalidMainPath   = errors.New("ko: invalid Main path: ko.main (or build.main if ko.main is not set) should be a relative path")
	errInvalidMainGoPath = errors.New("ko: invalid Main path: your path should point to a directory instead of a .go file")
)

// Pipe that build OCI compliant images with ko.
type Pipe struct{}

func (Pipe) String() string { return "ko" }
func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Ko) || len(ctx.Config.Kos) == 0
}

// Default sets the Pipes defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("kos")
	for i := range ctx.Config.Kos {
		ko := &ctx.Config.Kos[i]
		if ko.ID == "" {
			ko.ID = ctx.Config.ProjectName
		}

		if ko.Build == "" {
			ko.Build = ko.ID
		}

		build, err := findBuild(ctx, *ko)
		if err != nil {
			return err
		}

		if len(ko.Ldflags) == 0 {
			ko.Ldflags = build.Ldflags
		}

		if len(ko.Flags) == 0 {
			ko.Flags = build.Flags
		}

		if len(ko.Env) == 0 {
			ko.Env = build.Env
		}

		if ko.Main == "" {
			ko.Main = build.Main
		}

		if err := validateMainPath(ko.Main); err != nil {
			return err
		}

		if ko.WorkingDir == "" {
			ko.WorkingDir = build.Dir
		}

		if ko.BaseImage == "" {
			ko.BaseImage = chainguardStatic
		}

		if len(ko.Platforms) == 0 {
			ko.Platforms = []string{"linux/amd64"}
		}

		if len(ko.Tags) == 0 {
			ko.Tags = []string{"latest"}
		}

		switch ko.SBOM {
		case "cyclonedx", "go.version-m":
			deprecate.Notice(ctx, "kos.sbom")
			ko.SBOM = "none"
		case "":
			ko.SBOM = "spdx"
		}

		if ko.Repository != "" {
			ko.Repositories = append(ko.Repositories, ko.Repository)
			deprecate.Notice(ctx, "kos.repository")
		}

		if repo := ctx.Env["KO_DOCKER_REPO"]; repo != "" {
			ko.Repositories = []string{repo}
		}

		if len(ko.Repositories) == 0 {
			return errNoRepositories
		}

		ids.Inc(ko.ID)
	}
	return ids.Validate()
}

func (p Pipe) Run(ctx *context.Context) error {
	if ctx.Snapshot {
		// publish actually handles pushing to the local docker daemon when
		// snapshot is true.
		return p.Publish(ctx)
	}
	return nil
}

// Publish executes the Pipe.
func (Pipe) Publish(ctx *context.Context) error {
	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for _, ko := range ctx.Config.Kos {
		g.Go(func() error {
			disable, err := tmpl.New(ctx).Bool(ko.Disable)
			if err != nil {
				return err
			}
			if disable {
				return pipe.Skip("configuration is disabled")
			}
			return doBuild(ctx, ko)
		})
	}
	return g.Wait()
}

type buildOptions struct {
	importPath          string
	main                string
	flags               []string
	env                 []string
	imageRepos          []string
	workingDir          string
	platforms           []string
	baseImage           string
	labels              map[string]string
	annotations         map[string]string
	user                string
	tags                []string
	creationTime        *v1.Time
	koDataCreationTime  *v1.Time
	sbom                string
	SBOMDirectory       string
	ldflags             []string
	bare                bool
	preserveImportPaths bool
	baseImportPaths     bool
}

func (o *buildOptions) makeBuilder(ctx *context.Context) (*build.Caching, error) {
	buildOptions := []build.Option{
		build.WithConfig(map[string]build.Config{
			o.importPath: {
				Ldflags: o.ldflags,
				Flags:   o.flags,
				Main:    o.main,
				Env:     o.env,
			},
		}),
		build.WithPlatforms(o.platforms...),
		build.WithBaseImages(func(_ stdctx.Context, _ string) (name.Reference, build.Result, error) {
			ref, err := name.ParseReference(o.baseImage)
			if err != nil {
				return nil, nil, err
			}

			if cached, found := baseImages.Load(o.baseImage); found {
				return ref, cached.(build.Result), nil
			}

			if localImage, _ := daemon.Image(ref); localImage != nil {
				baseImages.Store(o.baseImage, localImage)
				return ref, localImage, err
			}

			desc, err := remote.Get(
				ref,
				remote.WithAuthFromKeychain(keychain),
			)
			if err != nil {
				return nil, nil, err
			}
			if desc.MediaType.IsImage() {
				img, err := desc.Image()
				baseImages.Store(o.baseImage, img)
				return ref, img, err
			}
			if desc.MediaType.IsIndex() {
				idx, err := desc.ImageIndex()
				baseImages.Store(o.baseImage, idx)
				return ref, idx, err
			}
			return nil, nil, fmt.Errorf("unexpected base image media type: %s", desc.MediaType)
		}),
	}
	if o.creationTime != nil {
		buildOptions = append(buildOptions, build.WithCreationTime(*o.creationTime))
	}
	if o.koDataCreationTime != nil {
		buildOptions = append(buildOptions, build.WithKoDataCreationTime(*o.koDataCreationTime))
	}
	for k, v := range o.labels {
		buildOptions = append(buildOptions, build.WithLabel(k, v))
	}
	for k, v := range o.annotations {
		buildOptions = append(buildOptions, build.WithAnnotation(k, v))
	}
	if o.user != "" {
		buildOptions = append(buildOptions, build.WithUser(o.user))
	}
	switch o.sbom {
	case "spdx":
		buildOptions = append(buildOptions, build.WithSPDX("devel"))
		if o.SBOMDirectory != "" {
			buildOptions = append(buildOptions, build.WithSBOMDir(o.SBOMDirectory))
		}
	case "none":
		buildOptions = append(buildOptions, build.WithDisabledSBOM())
	default:
		return nil, fmt.Errorf("unknown sbom type: %q", o.sbom)
	}

	b, err := build.NewGo(ctx, o.workingDir, buildOptions...)
	if err != nil {
		return nil, fmt.Errorf("newGo: %w", err)
	}
	return build.NewCaching(b)
}

func getLocalDomain(ko config.Ko) string {
	localDomain := "goreleaser.ko.local"
	if ko.LocalDomain != "" {
		localDomain = ko.LocalDomain
	}
	return localDomain
}

func doBuild(ctx *context.Context, ko config.Ko) error {
	opts, err := buildBuildOptions(ctx, ko)
	if err != nil {
		return err
	}

	b, err := opts.makeBuilder(ctx)
	if err != nil {
		return fmt.Errorf("makeBuilder: %w", err)
	}
	r, err := b.Build(ctx, opts.importPath)
	if err != nil {
		return fmt.Errorf("build: %w", err)
	}

	po := &options.PublishOptions{
		DockerRepo:          opts.imageRepos[0],
		Bare:                opts.bare,
		PreserveImportPaths: opts.preserveImportPaths,
		BaseImportPaths:     opts.baseImportPaths,
		Tags:                opts.tags,
		Local:               ctx.Snapshot,
		LocalDomain:         getLocalDomain(ko),
	}
	var p publish.Interface
	if ctx.Snapshot {
		p, err = publish.NewDaemon(
			options.MakeNamer(po),
			opts.tags,
			publish.WithLocalDomain(po.LocalDomain),
		)
	} else {
		p, err = publish.NewDefault(
			opts.imageRepos[0],
			publish.WithTags(opts.tags),
			publish.WithNamer(options.MakeNamer(po)),
			publish.WithAuthFromKeychain(keychain),
		)
	}
	if err != nil {
		return fmt.Errorf("newPublisher: %w", err)
	}
	defer func() { _ = p.Close() }()
	ref, err := p.Publish(ctx, r, opts.importPath)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	if err := p.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}

	ctx.Artifacts.Add(makeArtifact(
		ko.ID,
		ref.Name(),
		ref.Context().Digest(ref.Identifier()).DigestStr(),
	))

	if ctx.Snapshot || len(opts.imageRepos) == 1 {
		// do not copy images when snapshoting, as these images will be
		// local only anyway.
		return nil
	}

	src := ref.Name()
	for i := 1; i < len(opts.imageRepos); i++ {
		for _, tag := range opts.tags {
			dst := opts.imageRepos[i] + ":" + tag
			digest, err := copyImage(src, dst)
			if err != nil {
				return err
			}
			ctx.Artifacts.Add(makeArtifact(ko.ID, dst, digest))
		}
	}

	return nil
}

func findBuild(ctx *context.Context, ko config.Ko) (config.Build, error) {
	for _, build := range ctx.Config.Builds {
		if build.ID == ko.Build {
			return build, nil
		}
	}
	return config.Build{}, fmt.Errorf("no builds with id %q", ko.Build)
}

func buildBuildOptions(ctx *context.Context, cfg config.Ko) (*buildOptions, error) {
	localImportPath := cfg.Main

	dir := filepath.Clean(cfg.WorkingDir)
	if dir == "." {
		dir = ""
	}

	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName,
		Dir:  dir,
	}, localImportPath)
	if err != nil {
		return nil, fmt.Errorf(
			"ko: %s does not contain a valid local import path (%s) for directory (%s): %w",
			cfg.ID, localImportPath, cfg.WorkingDir, err,
		)
	}

	if len(pkgs) != 1 {
		return nil, fmt.Errorf(
			"ko: %s results in %d local packages, only 1 is expected",
			cfg.ID, len(pkgs),
		)
	}

	opts := &buildOptions{
		importPath:          pkgs[0].PkgPath,
		workingDir:          cfg.WorkingDir,
		bare:                cfg.Bare,
		preserveImportPaths: cfg.PreserveImportPaths,
		baseImportPaths:     cfg.BaseImportPaths,
		baseImage:           cfg.BaseImage,
		platforms:           cfg.Platforms,
		sbom:                cfg.SBOM,
		imageRepos:          cfg.Repositories,
		user:                cfg.User,
		SBOMDirectory:       cfg.SBOMDirectory,
	}

	tags, err := applyTemplate(ctx, cfg.Tags)
	if err != nil {
		return nil, err
	}
	opts.tags = removeEmpty(tags)

	if cfg.CreationTime != "" {
		creationTime, err := getTimeFromTemplate(ctx, cfg.CreationTime)
		if err != nil {
			return nil, err
		}
		opts.creationTime = creationTime
	}

	if cfg.KoDataCreationTime != "" {
		koDataCreationTime, err := getTimeFromTemplate(ctx, cfg.KoDataCreationTime)
		if err != nil {
			return nil, err
		}
		opts.koDataCreationTime = koDataCreationTime
	}

	if len(cfg.Labels) > 0 {
		labels, err := applyTemplateToMapValues(ctx, cfg.Labels)
		if err != nil {
			return nil, err
		}
		opts.labels = labels
	}

	if len(cfg.Annotations) > 0 {
		annotations, err := applyTemplateToMapValues(ctx, cfg.Annotations)
		if err != nil {
			return nil, err
		}
		opts.annotations = annotations
	}

	if len(cfg.Env) > 0 {
		env, err := applyTemplate(ctx, cfg.Env)
		if err != nil {
			return nil, err
		}
		opts.env = env
	}

	if len(cfg.Flags) > 0 {
		flags, err := applyTemplate(ctx, cfg.Flags)
		if err != nil {
			return nil, err
		}
		opts.flags = flags
	}

	if len(cfg.Ldflags) > 0 {
		ldflags, err := applyTemplate(ctx, cfg.Ldflags)
		if err != nil {
			return nil, err
		}
		opts.ldflags = ldflags
	}
	return opts, nil
}

func removeEmpty(strs []string) []string {
	var res []string
	for _, s := range strs {
		if strings.TrimSpace(s) == "" {
			continue
		}
		res = append(res, s)
	}
	return res
}

func applyTemplate(ctx *context.Context, templateable []string) ([]string, error) {
	templated := make([]string, 0, len(templateable))
	for _, t := range templateable {
		tlf, err := tmpl.New(ctx).Apply(t)
		if err != nil {
			return nil, err
		}
		templated = append(templated, tlf)
	}
	return templated, nil
}

func applyTemplateToMapValues(ctx *context.Context, templateable map[string]string) (map[string]string, error) {
	templated := make(map[string]string, len(templateable))
	for k, v := range templateable {
		tv, err := tmpl.New(ctx).Apply(v)
		if err != nil {
			return nil, err
		}
		templated[k] = tv
	}
	return templated, nil
}

func getTimeFromTemplate(ctx *context.Context, t string) (*v1.Time, error) {
	epoch, err := tmpl.New(ctx).Apply(t)
	if err != nil {
		return nil, err
	}

	seconds, err := strconv.ParseInt(epoch, 10, 64)
	if err != nil {
		return nil, err
	}
	return &v1.Time{Time: time.Unix(seconds, 0)}, nil
}

func validateMainPath(path string) error {
	// if the path is empty, it's probably fine as ko will use the default value
	if path == "" {
		return nil
	}
	if matched, _ := regexp.MatchString(`^\.?(\.\/[^\/]?.*)?$`, path); !matched {
		return errInvalidMainPath
	}
	// paths sure can have dots in them, but if the path ends in .go, it's probably a file that one misundertood as a valid value
	if strings.HasSuffix(path, ".go") {
		return errInvalidMainGoPath
	}
	return nil
}

func copyImage(src, dst string) (string, error) {
	log.WithField("src", src).
		WithField("dst", dst).
		Info("copying manifest")
	if err := crane.Copy(src, dst, crane.WithAuthFromKeychain(keychain)); err != nil {
		return "", fmt.Errorf("ko: could not copy %q to %q: %w", src, dst, err)
	}
	digest, err := crane.Digest(dst, crane.WithAuthFromKeychain(keychain))
	if err != nil {
		return "", fmt.Errorf("ko: could not get digest of %q: %w", dst, err)
	}
	return digest, nil
}

func makeArtifact(id, name, digest string) *artifact.Artifact {
	art := &artifact.Artifact{
		Type:  artifact.DockerManifest,
		Name:  name,
		Path:  name,
		Extra: map[string]any{},
	}
	if id != "" {
		art.Extra[artifact.ExtraID] = id
	}
	if digest != "" {
		art.Extra[artifact.ExtraDigest] = digest
	}
	return art
}
