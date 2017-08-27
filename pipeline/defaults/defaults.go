// Package defaults implements the Pipe interface providing default values
// for missing configuration.
package defaults

import (
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
)

// NameTemplate default name_template for the archive.
const NameTemplate = "{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"

// SnapshotNameTemplate represents the default format for snapshot release names.
const SnapshotNameTemplate = "SNAPSHOT-{{ .Commit }}"

// ChecksumNameTemplate is the default name_template for the checksum file.
const ChecksumNameTemplate = "{{ .ProjectName }}_{{ .Version }}_checksums.txt"

// Pipe for brew deployment
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Setting defaults"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	ctx.Config.Dist = "dist"
	if ctx.Config.Snapshot.NameTemplate == "" {
		ctx.Config.Snapshot.NameTemplate = SnapshotNameTemplate
	}
	if ctx.Config.Checksum.NameTemplate == "" {
		ctx.Config.Checksum.NameTemplate = ChecksumNameTemplate
	}
	if err := setReleaseDefaults(ctx); err != nil {
		return err
	}
	if ctx.Config.ProjectName == "" {
		ctx.Config.ProjectName = ctx.Config.Release.GitHub.Name
	}
	setBuildDefaults(ctx)
	if ctx.Config.Brew.Install == "" {
		var installs []string
		for _, build := range ctx.Config.Builds {
			if !isBrewBuild(build) {
				continue
			}
			installs = append(
				installs,
				fmt.Sprintf(`bin.install "%s"`, build.Binary),
			)
		}
		ctx.Config.Brew.Install = strings.Join(installs, "\n")
	}
	err := setArchiveDefaults(ctx)
	log.WithField("config", ctx.Config).Debug("defaults set")
	return err
}

func isBrewBuild(build config.Build) bool {
	for _, ignore := range build.Ignore {
		if ignore.Goos == "darwin" && ignore.Goarch == "amd64" {
			return false
		}
	}
	return contains(build.Goos, "darwin") && contains(build.Goarch, "amd64")
}

func contains(ss []string, s string) bool {
	for _, zs := range ss {
		if zs == s {
			return true
		}
	}
	return false
}

func setReleaseDefaults(ctx *context.Context) error {
	if ctx.Config.Release.GitHub.Name != "" {
		return nil
	}
	repo, err := remoteRepo()
	if err != nil {
		return fmt.Errorf("failed reading repo from git: %v", err.Error())
	}
	ctx.Config.Release.GitHub = repo
	return nil
}

func setBuildDefaults(ctx *context.Context) {
	for i, build := range ctx.Config.Builds {
		ctx.Config.Builds[i] = buildWithDefaults(ctx, build)
	}
	if len(ctx.Config.Builds) == 0 {
		ctx.Config.Builds = []config.Build{
			buildWithDefaults(ctx, ctx.Config.SingleBuild),
		}
	}
}

func buildWithDefaults(ctx *context.Context, build config.Build) config.Build {
	if build.Binary == "" {
		build.Binary = ctx.Config.Release.GitHub.Name
	}
	if build.Main == "" {
		build.Main = "."
	}
	if len(build.Goos) == 0 {
		build.Goos = []string{"linux", "darwin"}
	}
	if len(build.Goarch) == 0 {
		build.Goarch = []string{"amd64", "386"}
	}
	if len(build.Goarm) == 0 {
		build.Goarm = []string{"6"}
	}
	if build.Ldflags == "" {
		build.Ldflags = "-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}"
	}
	return build
}

func setArchiveDefaults(ctx *context.Context) error {
	if ctx.Config.Archive.NameTemplate == "" {
		ctx.Config.Archive.NameTemplate = NameTemplate
	}
	if ctx.Config.Archive.Format == "" {
		ctx.Config.Archive.Format = "tar.gz"
	}
	if len(ctx.Config.Archive.Files) == 0 {
		ctx.Config.Archive.Files = []string{
			"licence*",
			"LICENCE*",
			"license*",
			"LICENSE*",
			"readme*",
			"README*",
			"changelog*",
			"CHANGELOG*",
		}
	}
	return nil
}
