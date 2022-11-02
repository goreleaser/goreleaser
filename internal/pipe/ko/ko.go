// Package ko implements the pipe interface with the intent of
// building OCI compliant images with ko.
package ko

import (
	stdcontext "context"
	"fmt"
	"github.com/chrismellard/docker-credential-acr-env/pkg/credhelper"
	"github.com/google/go-containerregistry/pkg/authn/github"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"golang.org/x/tools/go/packages"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/ko/pkg/build"
	"github.com/google/ko/pkg/commands/options"
	"github.com/google/ko/pkg/publish"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
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
	for _, ko := range ctx.Config.Kos {
		if err := setConfigDefaults(&ko); err != nil {
			return err
		}
		ids.Inc(ko.ID)
	}
	return ids.Validate()
}

func setConfigDefaults(cfg *config.Ko) error {
	cfg.Push = true

	if cfg.ID == "" {
		cfg.ID = "default"
	}

	if cfg.BaseImage == "" {
		cfg.BaseImage = "cgr.dev/chainguard/static" // TODO: we can discuss on this
	}

	return nil
}

// Run executes the Pipe.
func (Pipe) Run(ctx *context.Context) error {
	g := semerrgroup.New(ctx.Parallelism)
	for _, ko := range ctx.Config.Kos {
		g.Go(doBuild(ctx, ko))
	}
	return g.Wait()
}

type buildOptions struct {
	ip                   string
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
	bo := []build.Option{
		build.WithConfig(map[string]build.Config{
			o.ip: {
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

			desc, err := remote.Get(ref,
				remote.WithAuthFromKeychain(keychain))
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
		bo = append(bo, build.WithSPDX("devel"))
	case "cyclonedx":
		bo = append(bo, build.WithCycloneDX())
	case "go.version-m":
		bo = append(bo, build.WithGoVersionSBOM())
	case "none":
		// don't do anything.
	default:
		return nil, fmt.Errorf("unknown sbom type: %q", o.sbom)
	}

	b, err := build.NewGo(ctx, o.workingDir, bo...)
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
		r, err := b.Build(ctxBackground, opts.ip)
		if err != nil {
			return fmt.Errorf("build: %v", err)
		}

		namer := options.MakeNamer(&options.PublishOptions{
			DockerRepo:          opts.dockerRepo,
			Bare:                opts.bare,
			PreserveImportPaths: opts.preserverImportPaths,
			BaseImportPaths:     opts.baseImportPaths,
		})

		p, err := publish.NewDefault(opts.dockerRepo,
			publish.WithTags(opts.tags),
			publish.WithNamer(namer),
			publish.WithAuthFromKeychain(authn.DefaultKeychain))
		if err != nil {
			return fmt.Errorf("newDefault: %v", err)
		}
		_, err = p.Publish(ctxBackground, r, opts.ip)
		if err != nil {
			return fmt.Errorf("publish: %v", err)
		}
		return nil
	}
}

func fromConfig(ctx *context.Context, cfg config.Ko) (*buildOptions, error) {
	var ldflags []string
	// find the matching build id with the Ko config
	buildID := cfg.Build
	for _, b := range ctx.Config.Builds {
		if b.ID == buildID {
			ldflags = append(ldflags, b.Ldflags...)
			cfg.WorkingDir = b.Dir
			cfg.Main = b.Main

			ft, err := applyTemplate(b.Flags, ctx)
			if err != nil {
				return nil, err
			}
			cfg.Flags = ft

			et, err := applyTemplate(b.Env, ctx)
			if err != nil {
				return nil, err
			}
			cfg.Env = et
		}
	}

	if cfg.WorkingDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}

		cfg.WorkingDir = wd
	}

	localImportPath := fmt.Sprint(".", string(filepath.Separator), ".")

	dir := filepath.Clean(cfg.WorkingDir)
	if dir == "." {
		dir = ""
	}

	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName, Dir: dir}, localImportPath)
	if err != nil {
		return nil, fmt.Errorf("'builds': %s does not contain a valid local import path (%s) for directory (%s): %w", cfg.ID, localImportPath, cfg.WorkingDir, err)
	}

	if len(pkgs) != 1 {
		return nil, fmt.Errorf("'builds': %s results in %d local packages, only 1 is expected", cfg.ID, len(pkgs))
	}

	opts := &buildOptions{
		ip:                   pkgs[0].PkgPath,
		workingDir:           cfg.WorkingDir,
		bare:                 cfg.Bare,
		preserverImportPaths: cfg.PreserveImportPaths,
		baseImportPaths:      cfg.BaseImportPaths,
		baseImage:            "cgr.dev/chainguard/static",
		platforms:            []string{"linux/amd64"},
		tags:                 []string{"latest"},
		sbom:                 "spdx",
		ldflags:              []string{},
	}

	if cfg.BaseImage != "" {
		opts.baseImage = cfg.BaseImage
	}

	if cfg.Platforms != nil {
		opts.platforms = cfg.Platforms
	}

	if cfg.Tags != nil {
		opts.tags = cfg.Tags
	}

	if cfg.SBOM != "" {
		opts.sbom = cfg.SBOM
	}

	if ctx.Env["KO_DOCKER_REPO"] != "" {
		opts.dockerRepo = ctx.Env["KO_DOCKER_REPO"]
	} else {
		opts.dockerRepo = cfg.Repository
	}

	if ctx.Env["COSIGN_REPOSITORY"] != "" {
		opts.cosignRepo = ctx.Env["COSIGN_REPOSITORY"]
	} else {
		opts.cosignRepo = cfg.CosignRepository
	}

	if len(cfg.LDFlags) != 0 {
		ll, err := applyTemplate(cfg.LDFlags, ctx)
		if err != nil {
			return nil, err
		}
		ldflags = append(ldflags, ll...)
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
