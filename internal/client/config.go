package client

import (
	"github.com/goreleaser/goreleaser/pkg/config"
)

func RepoFromRef(ref config.RepoRef) Repo {
	return Repo{
		Owner:  ref.Owner,
		Name:   ref.Name,
		Branch: ref.Branch,
	}
}
