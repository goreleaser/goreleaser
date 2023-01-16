// Package ko implements the pipe interface with the intent of
// building OCI compliant images with ko.
package ko

import (
	stdctx "context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sync"

	"github.com/awslabs/amazon-ecr-credential-helper/ecr-login"
	"github.com/chrismellard/docker-credential-acr-env/pkg/credhelper"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/github"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/ko/pkg/build"
	"github.com/google/ko/pkg/commands/options"
	"github.com/google/ko/pkg/publish"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
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

	errNoRepository = errors.New("ko: missing repository: please set either the repository field or a $KO_DOCKER_REPO environment variable")
)

// Pipe that build OCI compliant images with ko.
type Pipe struct{}

func (Pipe) String() string { return "ko" }
func (Pipe) Skip(ctx *context.Context) bool {
	return ctx.SkipKo || len(ctx.Config.Kos) == 0
}

// Default sets the Pipes defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("kos")
	for i := range ctx.Config.Kos {
		ko := &ctx.Config.Kos[i]
		if ko.ID == "" {
			ko.ID = "default"
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

		if ko.SBOM == "" {
			ko.SBOM = "spdx"
		}

		if repo := ctx.Env["KO_DOCKER_REPO"]; repo != "" {
			ko.Repository = repo
			ko.RepositoryFromEnv = true
		}

		if ko.Repository == "" {
			return errNoRepository
		}

		ids.Inc(ko.ID)
	}
	return ids.Validate()
}

// Publish executes the Pipe.
func (Pipe) Publish(ctx *context.Context) error {
	g := semerrgroup.New(ctx.Parallelism)
	for _, ko := range ctx.Config.Kos {
		g.Go(doBuild(ctx, ko))
	}
	return g.Wait()
}

type buildOptions struct {
	importPath          string
	main                string
	flags               []string
	env                 []string
	imageRepo           string
	fromEnv             bool
	workingDir          string
	platforms           []string
	baseImage           string
	tags                []string
	sbom                string
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
		build.WithBaseImages(func(ctx stdctx.Context, s string) (name.Reference, build.Result, error) {
			ref, err := name.ParseReference(o.baseImage)
			if err != nil {
				return nil, nil, err
			}

			if cached, found := baseImages.Load(o.baseImage); found {
				return ref, cached.(build.Result), nil
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
	switch o.sbom {
	case "spdx":
		buildOptions = append(buildOptions, build.WithSPDX("devel"))
	case "cyclonedx":
		buildOptions = append(buildOptions, build.WithCycloneDX())
	case "go.version-m":
		buildOptions = append(buildOptions, build.WithGoVersionSBOM())
	case "none":
		// don't do anything.
	default:
		return nil, fmt.Errorf("unknown sbom type: %q", o.sbom)
	}

	b, err := build.NewGo(ctx, o.workingDir, buildOptions...)
	if err != nil {
		return nil, fmt.Errorf("newGo: %w", err)
	}
	return build.NewCaching(b)
}

func doBuild(ctx *context.Context, ko config.Ko) func() error {
	return func() error {
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

		po := []publish.Option{publish.WithTags(opts.tags), publish.WithAuthFromKeychain(keychain)}

		var repo string
		if opts.fromEnv {
			repo = opts.imageRepo
		} else {
			// image resource's `repo` takes precedence if set, and selects the
			// `--bare` namer so the image is named exactly `repo`.
			repo = opts.imageRepo
			po = append(po, publish.WithNamer(options.MakeNamer(&options.PublishOptions{
				DockerRepo:          opts.imageRepo,
				Bare:                opts.bare,
				PreserveImportPaths: opts.preserveImportPaths,
				BaseImportPaths:     opts.baseImportPaths,
			})))
		}

		p, err := publish.NewDefault(repo, po...)
		if err != nil {
			return fmt.Errorf("newDefault: %w", err)
		}
		defer func() { _ = p.Close() }()
		if _, err = p.Publish(ctx, r, opts.importPath); err != nil {
			return fmt.Errorf("publish: %w", err)
		}
		if err := p.Close(); err != nil {
			return fmt.Errorf("close: %w", err)
		}
		return nil
	}
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
	localImportPath := "./..."

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
		tags:                cfg.Tags,
		imageRepo:           cfg.Repository,
		fromEnv:             cfg.RepositoryFromEnv,
	}

	tags, err := applyTemplate(ctx, cfg.Tags)
	if err != nil {
		return nil, err
	}
	cfg.Tags = tags

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

func applyTemplate(ctx *context.Context, templateable []string) ([]string, error) {
	var templated []string
	for _, t := range templateable {
		tlf, err := tmpl.New(ctx).Apply(t)
		if err != nil {
			return nil, err
		}
		templated = append(templated, tlf)
	}
	return templated, nil
}
