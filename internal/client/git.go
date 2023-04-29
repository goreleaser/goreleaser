package client

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"golang.org/x/crypto/ssh"
)

// DefaulGittSSHCommand used for git over SSH.
const DefaulGittSSHCommand = "ssh -i {{ .KeyPath }} -o StrictHostKeyChecking=accept-new -F /dev/null"

type gitClient struct{}

// NewGitUploadClient
func NewGitUploadClient(ctx *context.Context) (FileCreator, error) {
	return &gitClient{}, nil
}

// CreateFile implements FileCreator
func (*gitClient) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo Repo, content []byte, path string, message string) error {
	key, err := tmpl.New(ctx).Apply(repo.PrivateKey)
	if err != nil {
		return err
	}

	key, err = keyPath(key)
	if err != nil {
		return err
	}

	url, err := tmpl.New(ctx).Apply(repo.GitURL)
	if err != nil {
		return err
	}

	if url == "" {
		return pipe.Skip("aur.git_url is empty")
	}

	sshcmd, err := tmpl.New(ctx).WithExtraFields(tmpl.Fields{
		"KeyPath": key,
	}).Apply(firstNonEmpty(repo.GitSSHCommand, DefaulGittSSHCommand))
	if err != nil {
		return err
	}

	parent := filepath.Join(ctx.Config.Dist, "aur", "repos")
	cwd := filepath.Join(parent, repo.Name)

	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}

	env := []string{fmt.Sprintf("GIT_SSH_COMMAND=%s", sshcmd)}

	// TODO: check, clone might fail, repo might be out of date, etc
	// TODO: maybe also pass --depth=1?
	if err := runGitCmds(ctx, parent, env, [][]string{
		{"clone", url, repo.Name},
	}); err != nil {
		return fmt.Errorf("failed to setup local AUR repo: %w", err)
	}

	if err := runGitCmds(ctx, cwd, env, [][]string{
		// setup auth et al
		{"config", "--local", "user.name", commitAuthor.Name},
		{"config", "--local", "user.email", commitAuthor.Email},
		{"config", "--local", "commit.gpgSign", "false"},
		{"config", "--local", "init.defaultBranch", "master"},
	}); err != nil {
		return fmt.Errorf("failed to setup local AUR repo: %w", err)
	}

	if err := os.WriteFile(filepath.Join(ctx.Config.Dist, repo.Name, path), content, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	log.WithField("repo", url).WithField("name", repo.Name).Info("pushing")
	if err := runGitCmds(ctx, cwd, env, [][]string{
		{"add", "-A", "."},
		{"commit", "-m", message},
		{"push", "origin", "HEAD"},
	}); err != nil {
		return fmt.Errorf("failed to push %q (%q): %w", repo.Name, url, err)
	}

	return nil
}

func keyPath(key string) (string, error) {
	if key == "" {
		return "", pipe.Skip("aur.private_key is empty")
	}

	path := key

	_, err := ssh.ParsePrivateKey([]byte(key))
	if isPasswordError(err) {
		return "", fmt.Errorf("key is password-protected")
	}

	if err == nil {
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
		return "", fmt.Errorf("could not stat aur.private_key: %w", err)
	}

	// in any case, ensure the key has the correct permissions.
	if err := os.Chmod(path, 0o600); err != nil {
		return "", fmt.Errorf("failed to ensure aur.private_key permissions: %w", err)
	}

	return path, nil
}

func isPasswordError(err error) bool {
	var kerr *ssh.PassphraseMissingError
	return errors.As(err, &kerr)
}

func runGitCmds(ctx *context.Context, cwd string, env []string, cmds [][]string) error {
	for _, cmd := range cmds {
		args := append([]string{"-C", cwd}, cmd...)
		if _, err := git.Clean(git.RunWithEnv(ctx, env, args...)); err != nil {
			return fmt.Errorf("%q failed: %w", strings.Join(cmd, " "), err)
		}
	}
	return nil
}

func firstNonEmpty(s1, s2 string) string {
	if s1 != "" {
		return s1
	}
	return s2
}
