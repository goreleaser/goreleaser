// Package project sets "high level" defaults related to the project.
package project

import (
	"errors"
	"os/exec"
	"strings"

	"github.com/goreleaser/goreleaser/v2/internal/cargo"
	"github.com/goreleaser/goreleaser/v2/internal/git"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Pipe implements defaulter to set the project name.
type Pipe struct{}

func (Pipe) String() string {
	return "project name"
}

// Default set project defaults.
func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.ProjectName != "" {
		return nil
	}

	for _, candidate := range []string{
		cargoName(),
		ctx.Config.Release.GitHub.Name,
		ctx.Config.Release.GitLab.Name,
		ctx.Config.Release.Gitea.Name,
		moduleName(ctx),
		gitRemote(ctx),
	} {
		if candidate == "" {
			continue
		}
		ctx.Config.ProjectName = candidate
		return nil
	}

	return errors.New("couldn't guess project_name, please add it to your config")
}

func cargoName() string {
	cargo, err := cargo.Open("Cargo.toml")
	if err != nil {
		return ""
	}
	if n := cargo.Package.Name; n != "" {
		return n
	}
	return ""
}

func moduleName(ctx *context.Context) string {
	bts, err := exec.CommandContext(ctx, "go", "list", "-m").CombinedOutput()
	if err != nil {
		return ""
	}

	mod := strings.TrimSpace(string(bts))

	// this is the default module used when go runs without a go module.
	// https://pkg.go.dev/cmd/go@master#hdr-Package_lists_and_patterns
	if mod == "command-line-arguments" {
		return ""
	}

	parts := strings.Split(mod, "/")
	return strings.TrimSpace(parts[len(parts)-1])
}

func gitRemote(ctx *context.Context) string {
	repo, err := git.ExtractRepoFromConfig(ctx)
	if err != nil {
		return ""
	}
	if err := repo.CheckSCM(); err != nil {
		return ""
	}
	return repo.Name
}
