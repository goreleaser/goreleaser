package client

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestRepoFromRef(t *testing.T) {
	owner := "someowner"
	name := "somename"
	branch := "somebranch"
	token := "sometoken"

	ref := config.RepoRef{
		Owner:  owner,
		Name:   name,
		Branch: branch,
		Token:  token,
	}
	repo := RepoFromRef(ref)

	require.Equal(t, owner, repo.Owner)
	require.Equal(t, name, repo.Name)
	require.Equal(t, branch, repo.Branch)
}
