package semver

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/goreleaser/goreleaser/internal/tmpl"

	"github.com/Masterminds/semver/v3"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe is a global hook pipe.
type Pipe struct{}

// String is the name of this pipe.
func (Pipe) String() string {
	return "parsing tag"
}

// Run executes the hooks.
func (Pipe) Run(ctx *context.Context) error {
	currentVer, err := trimCurrentTag(ctx)
	if err != nil {
		return err
	}
	sv, err := semver.NewVersion(currentVer)
	if err != nil {
		return fmt.Errorf("failed to parse tag '%s' as semver: %w", currentVer, err)
	}
	ctx.Semver = context.Semver{
		Major:      sv.Major(),
		Minor:      sv.Minor(),
		Patch:      sv.Patch(),
		Prerelease: sv.Prerelease(),
		Metadata:   sv.Metadata(),
	}
	ctx.Version = sv.String()
	return nil
}

func trimCurrentTag(ctx *context.Context) (string, error) {
	if ctx.Config.Git.TagPrefixes == nil || len(ctx.Config.Git.TagPrefixes) < 1 {
		return ctx.Git.CurrentTag, nil
	}
	tpl := tmpl.New(ctx)
	prefixes := make([]string, len(ctx.Config.Git.TagPrefixes))
	for i, prefix := range ctx.Config.Git.TagPrefixes {
		eval, err := tpl.Apply(prefix)
		if err != nil {
			return ctx.Git.CurrentTag, err
		}
		prefixes[i] = eval
	}
	reg := regexp.MustCompile(strings.Join(prefixes, "|") + "(.+)")
	if !reg.MatchString(ctx.Git.CurrentTag) {
		return "", fmt.Errorf("tag '%s' has none of expected prefixes %+v", ctx.Git.CurrentTag, ctx.Config.Git.TagPrefixes)
	}
	return reg.ReplaceAllString(ctx.Git.CurrentTag, "${1}"), nil
}
