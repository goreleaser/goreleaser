package release

import (
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Preflight runs early release checks before the build pipeline, so that
// likely problems (e.g., the token lacks push access, or the tag is already
// published as an immutable release) are surfaced before expensive build
// steps run.
type Preflight struct{}

func (Preflight) String() string { return "release preflight checks" }

func (Preflight) Skip(ctx *context.Context) (bool, error) {
	// SkipTokenCheck is set by `goreleaser build`, which never publishes a
	// release; running the check there would add a needless SCM round-trip
	// (and could fail) for a purely local build.
	if ctx.SkipTokenCheck || skips.Any(ctx, skips.Publish) {
		return true, nil
	}
	return tmpl.New(ctx).Bool(ctx.Config.Release.Disable)
}

// Run checks that the configured SCM token has permission to create a release.
func (Preflight) Run(ctx *context.Context) error {
	cli, err := releaseClient(ctx)
	if err != nil {
		return err
	}
	return runPreflight(ctx, cli)
}

func runPreflight(ctx *context.Context, cli client.Client) error {
	checker, ok := cli.(client.ReleaseChecker)
	if !ok {
		return nil
	}
	if err := checker.CanRelease(ctx); err != nil {
		failOnError, terr := tmpl.New(ctx).Bool(ctx.Config.Release.Preflight.FailOnError)
		if terr != nil {
			return terr
		}
		if failOnError {
			return err
		}
		log.WithError(err).Warn("release preflight check failed, continuing anyway (set release.preflight.fail_on_error to abort)")
	}
	return nil
}
