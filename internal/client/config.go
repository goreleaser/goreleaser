package client

import (
	"github.com/goreleaser/goreleaser/pkg/config"
)

// RepoFromRef converts a config.RepoRef into a Repo.
func RepoFromRef(ref config.RepoRef) Repo {
	return Repo{
		Owner:         ref.Owner,
		Name:          ref.Name,
		Branch:        ref.Branch,
		GitURL:        ref.GitURL,
		GitSSHCommand: ref.GitSSHCommand,
		PrivateKey:    ref.PrivateKey,
	}
}

// TemplateRef templates a config.RepoFromRef
func TemplateRef(apply func(s string) (string, error), ref config.RepoRef) (config.RepoRef, error) {
	name, err := apply(ref.Name)
	if err != nil {
		return ref, err
	}
	owner, err := apply(ref.Owner)
	if err != nil {
		return ref, err
	}
	branch, err := apply(ref.Branch)
	if err != nil {
		return ref, err
	}
	git_url, err := apply(ref.GitURL)
	if err != nil {
		return ref, err
	}
	private_key, err := apply(ref.PrivateKey)
	if err != nil {
		return ref, err
	}
	git_ssh_command, err := apply(ref.GitSSHCommand)
	if err != nil {
		return ref, err
	}
	return config.RepoRef{
		Owner:         owner,
		Name:          name,
		Token:         ref.Token,
		Branch:        branch,
		GitURL:        git_url,
		GitSSHCommand: git_ssh_command,
		PrivateKey:    private_key,
	}, nil
}
