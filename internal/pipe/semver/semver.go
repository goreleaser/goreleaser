// Package semver handles semver parsing.
package semver

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Pipe parses the current Git tag as a semantic version.
type Pipe struct{}

// String is the name of this pipe.
func (Pipe) String() string {
	return "parsing tag"
}

// Run parses the tag and populates the semantic-version fields.
func (Pipe) Run(ctx *context.Context) error {
	sv, err := semver.NewVersion(ctx.Git.CurrentTag)
	if err != nil {
		if skips.Any(ctx, skips.Validate) {
			log.WithError(err).
				WithField("tag", ctx.Git.CurrentTag).
				Warn("current tag is not semver")
			return pipe.ErrSkipValidateEnabled
		}
		return fmt.Errorf("failed to parse tag '%s' as semver: %w", ctx.Git.CurrentTag, err)
	}
	ctx.Semver = context.Semver{
		Major:      sv.Major(),
		Minor:      sv.Minor(),
		Patch:      sv.Patch(),
		Prerelease: sv.Prerelease(),
	}
	return nil
}
