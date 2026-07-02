package client

import (
	"fmt"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestRepoFromRef(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	expected := config.RepoRef{
		Owner:  "owner",
		Name:   "name",
		Branch: "branch",
		Token:  "token",
		Git: config.GitRepoRef{
			URL:        "giturl",
			SSHCommand: "gitsshcommand",
			PrivateKey: "privatekey",
		},
	}
	t.Run("success", func(t *testing.T) {
		t.Parallel()
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
		t.Parallel()
		_, err := TemplateRef(func(s string) (string, error) {
			if s == "token" || s == "owner" {
				return "", fmt.Errorf("nope")
			}
			return s, nil
		}, expected)
		require.Error(t, err)
	})
	t.Run("fail name", func(t *testing.T) {
		t.Parallel()
		_, err := TemplateRef(func(s string) (string, error) {
			if s == "token" || s == "name" {
				return "", fmt.Errorf("nope")
			}
			return s, nil
		}, expected)
		require.Error(t, err)
	})
	t.Run("fail branch", func(t *testing.T) {
		t.Parallel()
		_, err := TemplateRef(func(s string) (string, error) {
			if s == "token" || s == "branch" {
				return "", fmt.Errorf("nope")
			}
			return s, nil
		}, expected)
		require.Error(t, err)
	})
	t.Run("fail giturl", func(t *testing.T) {
		t.Parallel()
		_, err := TemplateRef(func(s string) (string, error) {
			if s == "token" || s == "giturl" {
				return "", fmt.Errorf("nope")
			}
			return s, nil
		}, expected)
		require.Error(t, err)
	})
	t.Run("fail privatekey", func(t *testing.T) {
		t.Parallel()
		_, err := TemplateRef(func(s string) (string, error) {
			if s == "token" || s == "privatekey" {
				return "", fmt.Errorf("nope")
			}
			return s, nil
		}, expected)
		require.Error(t, err)
	})
}

func TestTemplateRefBranchDefault(t *testing.T) {
	t.Parallel()
	apply := func(s string) (string, error) {
		return strings.NewReplacer(
			"{{ .ProjectName }}", "foo",
			"{{ .Version }}", "1.2.3",
		).Replace(s), nil
	}

	t.Run("pull request disabled keeps branch empty", func(t *testing.T) {
		t.Parallel()
		ref, err := TemplateRef(apply, config.RepoRef{})
		require.NoError(t, err)
		require.Empty(t, ref.Branch)
	})

	t.Run("pull request enabled defaults to a per-release branch", func(t *testing.T) {
		t.Parallel()
		ref, err := TemplateRef(apply, config.RepoRef{
			PullRequest: config.PullRequest{Enabled: true},
		})
		require.NoError(t, err)
		require.Equal(t, "foo-1.2.3", ref.Branch)
	})

	t.Run("user-set branch is preserved", func(t *testing.T) {
		t.Parallel()
		ref, err := TemplateRef(apply, config.RepoRef{
			Branch:      "my-branch",
			PullRequest: config.PullRequest{Enabled: true},
		})
		require.NoError(t, err)
		require.Equal(t, "my-branch", ref.Branch)
	})
}
