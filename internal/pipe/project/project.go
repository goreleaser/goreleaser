// Package project sets "high level" defaults related to the project.
package project

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe implemens defaulter to set the project name.
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
		ctx.Config.Release.GitHub.Name,
		ctx.Config.Release.GitLab.Name,
		ctx.Config.Release.Gitea.Name,
		moduleName(),
		gitRemote(ctx),
	} {
		if candidate == "" {
			continue
		}
		ctx.Config.ProjectName = candidate
		return nil
	}

	return fmt.Errorf("couldn't guess project_name, please add it to your config")
}

func moduleName() string {
	bts, err := exec.Command("go", "list", "-m").CombinedOutput()
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
