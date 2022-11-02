package client

import (
	"fmt"
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

func TestTemplateRef(t *testing.T) {
	expected := config.RepoRef{
		Owner:  "owner",
		Name:   "name",
		Branch: "branch",
		Token:  "token",
	}
	t.Run("success", func(t *testing.T) {
		ref, err := TemplateRef(func(s string) (string, error) {
			if s == "token" {
				return "", fmt.Errorf("nope")
			}
			return s, nil
		}, expected)
		require.NoError(t, err)
		require.Equal(t, expected, ref)
	})

	t.Run("fail owner", func(t *testing.T) {
		_, err := TemplateRef(func(s string) (string, error) {
			if s == "token" || s == "owner" {
				return "", fmt.Errorf("nope")
			}
			return s, nil
		}, expected)
		require.Error(t, err)
	})
	t.Run("fail name", func(t *testing.T) {
		_, err := TemplateRef(func(s string) (string, error) {
			if s == "token" || s == "name" {
				return "", fmt.Errorf("nope")
			}
			return s, nil
		}, expected)
		require.Error(t, err)
	})
	t.Run("fail branch", func(t *testing.T) {
		_, err := TemplateRef(func(s string) (string, error) {
			if s == "token" || s == "branch" {
				return "", fmt.Errorf("nope")
			}
			return s, nil
		}, expected)
		require.Error(t, err)
	})
}
