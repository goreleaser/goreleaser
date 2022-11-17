package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/commitauthor"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"golang.org/x/crypto/ssh"
)

func PublishArtifactToGitURL(ctx *context.Context, artifacts []*artifact.Artifact, target config.RepoRef, repo_author config.CommitAuthor, commit_template string) error {
	package_name := artifacts[0].Name

	key, err := tmpl.New(ctx).Apply(target.PrivateKey)
	if err != nil {
		return err
	}

	key, err = keyPath(key)
	if err != nil {
		return err
	}

	url, err := tmpl.New(ctx).Apply(target.GitURL)
	if err != nil {
		return err
	}

	if url == "" {
		return pipe.Skip("brew.tap.git_url is empty")
	}

	sshcmd, err := tmpl.New(ctx).WithExtraFields(tmpl.Fields{
		"KeyPath": key,
	}).Apply(target.GitSSHCommand)
	if err != nil {
		return err
	}

	msg, err := tmpl.New(ctx).Apply(commit_template)
	if err != nil {
		return err
	}

	author, err := commitauthor.Get(ctx, repo_author)
	if err != nil {
		return err
	}

	parent := filepath.Join(ctx.Config.Dist, "brew", "repos")
	cwd := filepath.Join(parent, package_name)

	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}

	env := []string{fmt.Sprintf("GIT_SSH_COMMAND=%s", sshcmd)}

	if err := git.RunCmds(ctx, parent, env, [][]string{
		{"clone", url, package_name},
	}); err != nil {
		return fmt.Errorf("failed to setup local Brew repo: %w", err)
	}

	if err := git.RunCmds(ctx, cwd, env, [][]string{
		// setup auth et al
		{"config", "--local", "user.name", author.Name},
		{"config", "--local", "user.email", author.Email},
		{"config", "--local", "commit.gpgSign", "false"},
		{"config", "--local", "init.defaultBranch", "master"},
	}); err != nil {
		return fmt.Errorf("failed to setup local Brew repo: %w", err)
	}

	for _, pkg := range artifacts {
		bts, err := os.ReadFile(pkg.Path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", pkg.Name, err)
		}

		if err := os.WriteFile(filepath.Join(cwd, pkg.Name), bts, 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", pkg.Name, err)
		}
	}

	log.WithField("repo", url).WithField("name", package_name).Info("pushing")
	if err := git.RunCmds(ctx, cwd, env, [][]string{
		{"add", "-A", "."},
		{"commit", "-m", msg},
		{"push", "origin", "HEAD"},
	}); err != nil {
		return fmt.Errorf("failed to push %q (%q): %w", package_name, url, err)
	}

	return nil
}

func keyPath(key string) (string, error) {
	if key == "" {
		return "", pipe.Skip("brew.tap.private_key is empty")
	}

	path := key
	if _, err := ssh.ParsePrivateKey([]byte(key)); err == nil {
		// if it can be parsed as a valid private key, we write it to a
		// temp file and use that path on GIT_SSH_COMMAND.
		f, err := os.CreateTemp("", "id_*")
		if err != nil {
			return "", fmt.Errorf("failed to store private key: %w", err)
		}
		defer f.Close()

		// the key needs to EOF at an empty line, seems like github actions
		// is somehow removing them.
		if !strings.HasSuffix(key, "\n") {
			key += "\n"
		}

		if _, err := io.WriteString(f, key); err != nil {
			return "", fmt.Errorf("failed to store private key: %w", err)
		}
		if err := f.Close(); err != nil {
			return "", fmt.Errorf("failed to store private key: %w", err)
		}
		path = f.Name()
	}

	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("could not stat brew.tap.private_key: %w", err)
	}

	// in any case, ensure the key has the correct permissions.
	if err := os.Chmod(path, 0o600); err != nil {
		return "", fmt.Errorf("failed to ensure brew.tap.private_key permissions: %w", err)
	}

	return path, nil
}
