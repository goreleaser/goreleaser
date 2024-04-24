package client

import "github.com/goreleaser/goreleaser/pkg/config"

const prFooter = "###### Automated with [GoReleaser](https://goreleaser.com)"

// RepoFromRef converts a config.RepoRef into a Repo.
func RepoFromRef(ref config.RepoRef) Repo {
	return Repo{
		Owner:         ref.Owner,
		Name:          ref.Name,
		Branch:        ref.Branch,
		GitURL:        ref.Git.URL,
		GitSSHCommand: ref.Git.SSHCommand,
		PrivateKey:    ref.Git.PrivateKey,
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
	gitURL, err := apply(ref.Git.URL)
	if err != nil {
		return ref, err
	}
	privateKey, err := apply(ref.Git.PrivateKey)
	if err != nil {
		return ref, err
	}
	return config.RepoRef{
		Owner:       owner,
		Name:        name,
		Token:       ref.Token,
		Branch:      branch,
		PullRequest: ref.PullRequest,
		Git: config.GitRepoRef{
			URL:        gitURL,
			PrivateKey: privateKey,
			SSHCommand: ref.Git.SSHCommand,
		},
	}, nil
}
