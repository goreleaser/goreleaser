package notary

import (
	"fmt"
	"strings"
	"time"

	"github.com/anchore/quill/quill"
	"github.com/anchore/quill/quill/notary"
	"github.com/anchore/quill/quill/pki/load"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

type MacOS struct{}

func (MacOS) String() string { return "sign & notarize macOS binaries" }

func (MacOS) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Notarize) || len(ctx.Config.Notarize.MacOS) == 0
}

func (MacOS) Default(ctx *context.Context) error {
	for i := range ctx.Config.Notarize.MacOS {
		n := &ctx.Config.Notarize.MacOS[i]
		if n.Notarize.Timeout == 0 {
			n.Notarize.Timeout = 10 * time.Minute
		}
		if len(n.IDs) == 0 {
			n.IDs = []string{ctx.Config.ProjectName}
		}
	}
	return nil
}

func (MacOS) Run(ctx *context.Context) error {
	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for _, cfg := range ctx.Config.Notarize.MacOS {
		g.Go(func() error {
			return signAndNotarize(ctx, cfg)
		})
	}
	return g.Wait()
}

func signAndNotarize(ctx *context.Context, cfg config.MacOSSignNotarize) error {
	ok, err := tmpl.New(ctx).Bool(cfg.Enabled)
	if err != nil {
		return fmt.Errorf("notarize: macos: %w", err)
	}
	if !ok {
		return pipe.Skip("disabled")
	}

	if err := tmpl.New(ctx).ApplyAll(
		&cfg.Sign.Certificate,
		&cfg.Sign.Password,
		&cfg.Notarize.Key,
		&cfg.Notarize.KeyID,
		&cfg.Notarize.IssuerID,
	); err != nil {
		return fmt.Errorf("notarize: macos: %w", err)
	}

	p12, err := load.P12(cfg.Sign.Certificate, cfg.Sign.Password)
	if err != nil {
		return fmt.Errorf("notarize: macos: %w", err)
	}

	filters := []artifact.Filter{
		artifact.ByGoos("darwin"),
		artifact.Or(
			artifact.ByType(artifact.Binary),
			artifact.ByType(artifact.UniversalBinary),
		),
	}
	if len(cfg.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(cfg.IDs...))
	}
	binaries := ctx.Artifacts.Filter(artifact.And(filters...))
	if len(binaries.List()) == 0 {
		return pipe.Skipf("no darwin binaries found with ids: %s", strings.Join(cfg.IDs, ", "))
	}

	for _, bin := range binaries.List() {
		signCfg, err := quill.NewSigningConfigFromP12(bin.Path, *p12, true)
		if err != nil {
			return fmt.Errorf("notarize: macos: %s: %w", bin.Path, err)
		}
		signCfg = signCfg.WithTimestampServer("http://timestamp.apple.com/ts01")

		log.WithField("binary", bin.Path).Info("signing")
		if err := quill.Sign(*signCfg); err != nil {
			return fmt.Errorf("notarize: macos: %s: %w", bin.Path, err)
		}

		notarizeCfg := quill.NewNotarizeConfig(
			cfg.Notarize.IssuerID,
			cfg.Notarize.KeyID,
			cfg.Notarize.Key,
		).WithStatusConfig(notary.StatusConfig{
			Timeout: cfg.Notarize.Timeout,
			Poll:    10,
			Wait:    cfg.Notarize.Wait,
		})

		if cfg.Notarize.Wait {
			log.WithField("binary", bin.Path).Info("notarizing and waiting - this might take a while")
		} else {
			log.WithField("binary", bin.Path).Info("sending notarize request")
		}
		status, err := quill.Notarize(bin.Path, *notarizeCfg)
		if err != nil {
			return fmt.Errorf("notarize: macos: %s: %w", bin.Path, err)
		}

		switch status {
		case notary.AcceptedStatus:
			log.WithField("binary", bin.Path).Info("notarized")
		case notary.InvalidStatus:
			return fmt.Errorf("notarize: macos: %s: invalid", bin.Path)
		case notary.RejectedStatus:
			return fmt.Errorf("notarize: macos: %s: rejected", bin.Path)
		case notary.TimeoutStatus:
			log.WithField("binary", bin.Path).Info("notarize timeout")
		default:
			log.WithField("binary", bin.Path).Info("notarize still pending")
		}

	}

	if err := binaries.Refresh(); err != nil {
		return fmt.Errorf("notarize: macos: refresh artifacts: %w", err)
	}
	return nil
}
