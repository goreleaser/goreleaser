// Package ko implements the pipe interface with the intent of
// building OCI compliant images with ko.
package ko

import (
	stdcontext "context"
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

		if ko.WorkingDir == "" {
			ko.WorkingDir = "."
		}

		if ko.BaseImage == "" {
			ko.BaseImage = chainguardStatic
		}

		if repo := ctx.Env["KO_DOCKER_REPO"]; repo != "" {
			ko.Repository = repo
		}

		if repo := ctx.Env["COSIGN_REPOSITORY"]; repo != "" {
			ko.CosignRepository = repo
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
	importPath           string
	main                 string
	flags                []string
	env                  []string
	workingDir           string
	dockerRepo           string
	cosignRepo           string
	platforms            []string
	baseImage            string
	tags                 []string
	sbom                 string
	ldflags              []string
	bare                 bool
	preserverImportPaths bool
	baseImportPaths      bool
}

var baseImages sync.Map

var (
	amazonKeychain authn.Keychain = authn.NewKeychainFromHelper(ecr.NewECRHelper(ecr.WithLogger(io.Discard)))
	azureKeychain  authn.Keychain = authn.NewKeychainFromHelper(credhelper.NewACRCredentialsHelper())
	keychain                      = authn.NewMultiKeychain(
		amazonKeychain,
		authn.DefaultKeychain,
		google.Keychain,
		github.Keychain,
		azureKeychain,
	)
)

func (o *buildOptions) makeBuilder(ctx stdcontext.Context) (*build.Caching, error) {
	buildOptions := []build.Option{
		build.WithConfig(map[string]build.Config{
			o.importPath: {
				Ldflags: o.ldflags,
				Main:    o.main,
				Flags:   o.flags,
				Env:     o.env,
			},
		}),
		build.WithPlatforms(o.platforms...),
		build.WithBaseImages(func(ctx stdcontext.Context, s string) (name.Reference, build.Result, error) {
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
		return nil, fmt.Errorf("newGo: %v", err)
	}
	return build.NewCaching(b)
}

func doBuild(ctx *context.Context, ko config.Ko) func() error {
	return func() error {
		ctxBackground, cancel := stdcontext.WithCancel(stdcontext.Background())
		defer cancel()

		opts, err := fromConfig(ctx, ko)
		if err != nil {
			return err
		}

		b, err := opts.makeBuilder(ctxBackground)
		if err != nil {
			return fmt.Errorf("makeBuilder: %v", err)
		}
		r, err := b.Build(ctxBackground, opts.importPath)
		if err != nil {
			return fmt.Errorf("build: %v", err)
		}

		namer := options.MakeNamer(&options.PublishOptions{
			DockerRepo:          opts.dockerRepo,
			Bare:                opts.bare,
			PreserveImportPaths: opts.preserverImportPaths,
			BaseImportPaths:     opts.baseImportPaths,
		})

		p, err := publish.NewDefault(
			opts.dockerRepo,
			publish.WithTags(opts.tags),
			publish.WithNamer(namer),
			publish.WithAuthFromKeychain(authn.DefaultKeychain),
		)
		if err != nil {
			return fmt.Errorf("newDefault: %v", err)
		}
		defer func() { _ = p.Close() }()
		if _, err = p.Publish(ctxBackground, r, opts.importPath); err != nil {
			return fmt.Errorf("publish: %v", err)
		}
		if err := p.Close(); err != nil {
			return fmt.Errorf("close: %v", err)
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
	return config.Build{}, fmt.Errorf("no builds with id %s", ko.Build)
}

func fromConfig(ctx *context.Context, cfg config.Ko) (*buildOptions, error) {
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
		importPath:           pkgs[0].PkgPath,
		workingDir:           cfg.WorkingDir,
		bare:                 cfg.Bare,
		preserverImportPaths: cfg.PreserveImportPaths,
		baseImportPaths:      cfg.BaseImportPaths,
		baseImage:            cfg.BaseImage,
		platforms:            cfg.Platforms,
		tags:                 cfg.Tags,
		sbom:                 cfg.SBOM,
		ldflags:              cfg.Ldflags,
		dockerRepo:           cfg.Repository,
		cosignRepo:           cfg.CosignRepository,
	}

	if len(cfg.Env) > 0 {
		env, err := applyTemplate(cfg.Env, ctx)
		if err != nil {
			return nil, err
		}
		opts.env = env
	}

	if len(cfg.Flags) > 0 {
		flags, err := applyTemplate(cfg.Flags, ctx)
		if err != nil {
			return nil, err
		}
		opts.flags = flags
	}

	if len(cfg.Ldflags) > 0 {
		ldflags, err := applyTemplate(cfg.Ldflags, ctx)
		if err != nil {
			return nil, err
		}
		opts.ldflags = ldflags
	}
	return opts, nil
}

func applyTemplate(templateable []string, ctx *context.Context) ([]string, error) {
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
