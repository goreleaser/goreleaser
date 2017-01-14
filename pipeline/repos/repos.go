package repos

import (
	"context"
	"strings"

	"github.com/goreleaser/releaser/context"
)

// Pipe for brew deployment
type Pipe struct{}

// Name of the pipe
func (Pipe) Name() string {
	return "Repos"
}

// Run the pipe
func (Pipe) Run(context *context.Context) (err error) {
	owner, name := split(context.Config.Repo)
	context.Repo.Owner = owner
	context.Repo.Name = name
	owner, name = split(context.Config.Brew.Repo)
	context.BrewRepo.Owner = owner
	context.BrewRepo.Name = name
	return
}

func split(pair string) (string, string) {
	parts := strings.Split(pair, "/")
	return parts[0], parts[1]
}
