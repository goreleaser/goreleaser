package semver

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/goreleaser/goreleaser/internal/deprecate"
	"github.com/goreleaser/goreleaser/internal/pipe"
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
	sv, err := semver.NewVersion(ctx.Git.CurrentTag)
	if err != nil {
		if ctx.Snapshot || ctx.SkipValidate {
			deprecate.NoticeCustom(
				ctx,
				"skipping-semver-validations",
				fmt.Sprintf("'%s' is not SemVer-compatible and may cause other issues in the pipeline, check {{ .URL }} for more info", ctx.Git.CurrentTag),
			)
		}
		if ctx.Snapshot {
			return pipe.ErrSnapshotEnabled
		}
		if ctx.SkipValidate {
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
