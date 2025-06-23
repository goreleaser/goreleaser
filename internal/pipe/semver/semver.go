// Package semver handles semver parsing.
package semver

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Pipe is a global hook pipe.
type Pipe struct{}

// String is the name of this pipe.
func (Pipe) String() string {
	return "parsing tag"
}

func getTag(ctx *context.Context) (string, error) {
	if ctx.Config.Semver.Version == "" {
		return ctx.Git.CurrentTag, nil
	}
	tag, err := tmpl.New(ctx).Apply(ctx.Config.Semver.Version)
	if err != nil {
		return "", fmt.Errorf("semver.process: %w", err)
	}
	log.Infof("using %q instead of %q", tag, ctx.Git.CurrentTag)
	return tag, nil
}

// Run executes the hooks.
func (Pipe) Run(ctx *context.Context) error {
	tag, err := getTag(ctx)
	if err != nil {
		return err
	}

	sv, err := semver.NewVersion(tag)
	if err != nil {
		return fmt.Errorf("failed to parse tag '%s' as semver: %w", tag, err)
	}
	ctx.Semver = context.Semver{
		Major:      sv.Major(),
		Minor:      sv.Minor(),
		Patch:      sv.Patch(),
		Prerelease: sv.Prerelease(),
	}
	return nil
}
