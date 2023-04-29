package client

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"golang.org/x/crypto/ssh"
)

// DefaulGitSSHCommand used for git over SSH.
const DefaulGitSSHCommand = `ssh -i "{{ .KeyPath }}" -o StrictHostKeyChecking=accept-new -F /dev/null`

var cloneLock = cloneGlobalLock{
	l:     sync.Mutex{},
	repos: map[string]bool{},
}

type gitClient struct {
	branch string
}

// NewGitUploadClient
func NewGitUploadClient(branch string) FileCreator {
	return &gitClient{
		branch: branch,
	}
}

// CreateFile implements FileCreator
func (g *gitClient) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo Repo, content []byte, path string, message string) error {
	url, err := tmpl.New(ctx).Apply(repo.GitURL)
	if err != nil {
		return fmt.Errorf("git: failed to template git url: %w", err)
	}

	if url == "" {
		return pipe.Skip("url is empty")
	}

	repo.Name = firstNonEmpty(repo.Name, nameFromURL(url))

	key, err := tmpl.New(ctx).Apply(repo.PrivateKey)
	if err != nil {
		return fmt.Errorf("git: failed to template private key: %w", err)
	}

	key, err = keyPath(key)
	if err != nil {
		return err
	}

	sshcmd, err := tmpl.New(ctx).WithExtraFields(tmpl.Fields{
		"KeyPath": key,
	}).Apply(firstNonEmpty(repo.GitSSHCommand, DefaulGitSSHCommand))
	if err != nil {
		return fmt.Errorf("git: failed to template ssh command: %w", err)
	}

	parent := filepath.Join(ctx.Config.Dist, "git")
	cwd := filepath.Join(parent, repo.Name)
	env := []string{fmt.Sprintf("GIT_SSH_COMMAND=%s", sshcmd)}

	if err := cloneLock.clone(url, func() error {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return fmt.Errorf("git: failed to create parent: %w", err)
		}

		if err := runGitCmds(ctx, parent, env, [][]string{
			{"clone", url, repo.Name},
		}); err != nil {
			return fmt.Errorf("git: failed to clone local repository: %w", err)
		}

		if err := runGitCmds(ctx, cwd, env, [][]string{
			{"config", "--local", "user.name", commitAuthor.Name},
			{"config", "--local", "user.email", commitAuthor.Email},
			{"config", "--local", "commit.gpgSign", "false"},
			{"config", "--local", "init.defaultBranch", firstNonEmpty(g.branch, "master")},
		}); err != nil {
			return fmt.Errorf("git: failed to setup local repository: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	location := filepath.Join(cwd, path)
	log.WithField("path", location).Info("writing")
	if err := os.MkdirAll(filepath.Dir(location), 0o755); err != nil {
		return fmt.Errorf("failed to create parent dirs for %s: %w", path, err)
	}
	if err := os.WriteFile(location, content, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	log.WithField("repo", url).WithField("name", repo.Name).Info("pushing")
	if err := runGitCmds(ctx, cwd, env, [][]string{
		{"add", "-A", "."},
		{"commit", "-m", message},
		{"push", "origin", "HEAD"},
	}); err != nil {
		return fmt.Errorf("git: failed to push %q (%q): %w", repo.Name, url, err)
	}

	return nil
}

func keyPath(key string) (string, error) {
	if key == "" {
		return "", pipe.Skip("private_key is empty")
	}

	path := key

	_, err := ssh.ParsePrivateKey([]byte(key))
	if isPasswordError(err) {
		return "", fmt.Errorf("git: key is password-protected")
	}

	if err == nil {
		// if it can be parsed as a valid private key, we write it to a
		// temp file and use that path on GIT_SSH_COMMAND.
		f, err := os.CreateTemp("", "id_*")
		if err != nil {
			return "", fmt.Errorf("git: failed to store private key: %w", err)
		}
		defer f.Close()

		// the key needs to EOF at an empty line, seems like github actions
		// is somehow removing them.
		if !strings.HasSuffix(key, "\n") {
			key += "\n"
		}

		if _, err := io.WriteString(f, key); err != nil {
			return "", fmt.Errorf("git: failed to store private key: %w", err)
		}
		if err := f.Close(); err != nil {
			return "", fmt.Errorf("git: failed to store private key: %w", err)
		}
		path = f.Name()
	}

	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("git: could not stat private_key: %w", err)
	}

	// in any case, ensure the key has the correct permissions.
	if err := os.Chmod(path, 0o600); err != nil {
		return "", fmt.Errorf("git: failed to ensure private_key permissions: %w", err)
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

func nameFromURL(url string) string {
	return strings.TrimSuffix(url[strings.LastIndex(url, "/")+1:], ".git")
}

type cloneGlobalLock struct {
	l     sync.Mutex
	repos map[string]bool
}

func (c *cloneGlobalLock) clone(url string, fn func() error) error {
	c.l.Lock()
	defer c.l.Unlock()

	if c.repos[url] {
		return nil
	}

	c.repos[url] = true
	return fn()
}
