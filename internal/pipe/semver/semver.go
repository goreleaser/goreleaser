package semver

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Pipe is a global hook pipe.
type Pipe struct{}

// String is the name of this pipe.
func (Pipe) String() string {
	return "parsing tag"
}

// Run executes the hooks.
func (Pipe) Run(ctx *context.Context) error {
	sv, err := semver.NewVersion(handleCalver(ctx.Git.CurrentTag))
	if err != nil {
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

func handleCalver(v string) string {
	var parts []string
	for i, part := range strings.SplitN(strings.TrimPrefix(v, "v"), ".", 3) {
		if i == 0 && len(part) != 4 {
			// first part is a year, not calver
			return v
		}
		parts = append(parts, strings.TrimPrefix(part, "0"))
	}

	result := strings.Join(parts, ".")
	if strings.HasPrefix(v, "v") {
		result = "v" + result
	}
	return result
}
