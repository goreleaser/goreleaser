package client

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/commitauthor"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"golang.org/x/crypto/ssh"
)

const defaultSSHCommand = "ssh -i {{ .KeyPath }} -o StrictHostKeyChecking=accept-new -F /dev/null"

type gitClient struct{}

func NewGitUploadClient(ctx *context.Context) (Client, error) {
	return &gitClient{}, nil
}

func (gc *gitClient) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo Repo, content []byte, path, message string) (err error) {
	key, err := tmpl.New(ctx).Apply(repo.PrivateKey)
	if err != nil {
		return err
	}

	key, err = keyPath(key)
	if err != nil {
		return err
	}

	var gitSSHCommand string
	if len(repo.GitSSHCommand) > 0 {
		gitSSHCommand = repo.GitSSHCommand
	} else {
		gitSSHCommand = defaultSSHCommand
	}

	sshCmd, err := tmpl.New(ctx).WithExtraFields(tmpl.Fields{
		"KeyPath": key,
	}).Apply(gitSSHCommand)
	if err != nil {
		return err
	}

	url, err := tmpl.New(ctx).Apply(repo.GitURL)
	if err != nil {
		return err
	}

	if url == "" {
		return fmt.Errorf("git_url is empty")
	}

	msg, err := tmpl.New(ctx).Apply(message)
	if err != nil {
		return err
	}

	author, err := commitauthor.Get(ctx, commitAuthor)
	if err != nil {
		return err
	}

	env := []string{fmt.Sprintf("GIT_SSH_COMMAND=%s", sshCmd)}

	repoName := strings.TrimSuffix(url[strings.LastIndex(url, "/")+1:], ".git")

	_, err = os.Stat(filepath.Join(ctx.Config.Dist, repoName))

	// Only clone if this repo has not been cloned before
	if os.IsNotExist(err) {
		if err := runCmds(ctx, ctx.Config.Dist, env, [][]string{
			{"clone", url},
		}); err != nil {
			return fmt.Errorf("failed to setup local Brew repo: %w", err)
		}
	} else if err != nil {
		return err
	}

	if err := runCmds(ctx, filepath.Join(ctx.Config.Dist, repoName), env, [][]string{
		// setup auth et al
		{"config", "--local", "user.name", author.Name},
		{"config", "--local", "user.email", author.Email},
		{"config", "--local", "commit.gpgSign", "false"},
		{"config", "--local", "init.defaultBranch", "master"},
	}); err != nil {
		return fmt.Errorf("failed to setup local Brew repo: %w", err)
	}

	if err := os.WriteFile(filepath.Join(ctx.Config.Dist, repoName, path), content, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	log.WithField("repo", url).WithField("name", path).Info("pushing")
	if err := runCmds(ctx, filepath.Join(ctx.Config.Dist, repoName), env, [][]string{
		{"add", "-A", "."},
		{"commit", "-m", msg},
		{"push", "origin", "HEAD"},
	}); err != nil {
		return fmt.Errorf("failed to push %q (%q): %w", path, url, err)
	}

	return nil
}

func runCmds(ctx *context.Context, cwd string, env []string, cmds [][]string) error {
	for _, cmd := range cmds {
		args := append([]string{"-C", cwd}, cmd...)
		if _, err := git.Clean(git.RunWithEnv(ctx, env, args...)); err != nil {
			return fmt.Errorf("%q failed: %w", strings.Join(cmd, " "), err)
		}
	}
	return nil
}

func keyPath(key string) (string, error) {
	if key == "" {
		return "", pipe.Skip("private_key is empty")
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
		return "", fmt.Errorf("could not stat private_key: %w", err)
	}

	// in any case, ensure the key has the correct permissions.
	if err := os.Chmod(path, 0o600); err != nil {
		return "", fmt.Errorf("failed to ensure private_key permissions: %w", err)
	}

	return path, nil
}

func (gc *gitClient) CloseMilestone(ctx *context.Context, repo Repo, title string) (err error) {
	return ErrNotImplemented
}

func (gc *gitClient) CreateRelease(ctx *context.Context, body string) (releaseID string, err error) {
	return "", ErrNotImplemented
}

func (gc *gitClient) ReleaseURLTemplate(ctx *context.Context) (string, error) {
	return "", ErrNotImplemented
}

func (gc *gitClient) Upload(ctx *context.Context, releaseID string, artifact *artifact.Artifact, file *os.File) (err error) {
	return ErrNotImplemented
}

func (gc *gitClient) GetDefaultBranch(ctx *context.Context, repo Repo) (string, error) {
	return "", ErrNotImplemented
}

func (gc *gitClient) Changelog(ctx *context.Context, repo Repo, prev, current string) (string, error) {
	return "", ErrNotImplemented
}
