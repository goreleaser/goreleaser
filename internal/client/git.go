package client

import (
	"cmp"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/git"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/redact"
	"github.com/goreleaser/goreleaser/v2/internal/retryx"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"golang.org/x/crypto/ssh"
)

var gil sync.Mutex

// DefaultGitSSHCommand used for git over SSH.
const DefaultGitSSHCommand = `ssh -i "{{ .KeyPath }}" -o StrictHostKeyChecking=accept-new -F /dev/null`

type gitClient struct {
	branch string
}

// NewGitUploadClient creates a new git client.
func NewGitUploadClient(branch string) FilesCreator {
	return &gitClient{
		branch: branch,
	}
}

// CreateFiles implements FilesCreator.
func (g *gitClient) CreateFiles(
	ctx *context.Context,
	commitAuthor config.CommitAuthor,
	repo Repo,
	message string,
	files []RepoFile,
) (err error) {
	gil.Lock()
	defer gil.Unlock()

	url, err := tmpl.New(ctx).Apply(repo.GitURL)
	if err != nil {
		return fmt.Errorf("git: failed to template git url: %w", err)
	}

	if url == "" {
		return pipe.Skip("url is empty")
	}

	repo.Name = cmp.Or(repo.Name, nameFromURL(url))

	key, err := tmpl.New(ctx).Apply(repo.PrivateKey)
	if err != nil {
		return fmt.Errorf("git: failed to template private key: %w", err)
	}

	key, cleanupKey, err := keyPath(key)
	if err != nil {
		return err
	}
	if cleanupKey != nil {
		defer func() {
			err = errors.Join(err, cleanupKey())
		}()
	}

	sshcmd, err := tmpl.New(ctx).WithExtraFields(tmpl.Fields{
		"KeyPath": key,
	}).Apply(cmp.Or(repo.GitSSHCommand, DefaultGitSSHCommand))
	if err != nil {
		return fmt.Errorf("git: failed to template ssh command: %w", err)
	}

	parent := filepath.Join(ctx.Config.Dist, "git")
	name := checkoutDirName(repo.Name, url, g.branch)
	cwd := filepath.Join(parent, name)
	env := []string{fmt.Sprintf("GIT_SSH_COMMAND=%s", sshcmd)}

	if _, err := os.Stat(cwd); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return fmt.Errorf("git: failed to create parent: %w", err)
		}

		if err := cloneRepo(ctx, parent, url, name, env); err != nil {
			return err
		}

		if g.branch != "" {
			if err := runGitCmd(ctx, cwd, env, "checkout", g.branch); err != nil {
				if err := runGitCmd(ctx, cwd, env, "checkout", "-b", g.branch); err != nil {
					return fmt.Errorf("git: could not checkout branch %s: %w", g.branch, err)
				}
			}
		}
	}

	for _, file := range files {
		location := filepath.Join(cwd, file.Path)
		log.WithField("path", location).Info("writing")
		if err := os.MkdirAll(filepath.Dir(location), 0o755); err != nil {
			return fmt.Errorf("failed to create parent dirs for %s: %w", file.Path, err)
		}
		if err := os.WriteFile(location, file.Content, 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", file.Path, err)
		}
		log.
			WithField("repository", url).
			WithField("name", repo.Name).
			WithField("file", file.Path).
			Info("pushing")
	}

	if err := runGitCmd(ctx, cwd, env, "add", "-A", "."); err != nil {
		return fmt.Errorf("git: failed to add files %q (%q): %w", repo.Name, url, err)
	}
	if err := runGitCmdWith(
		ctx, cwd, env,
		commitConfigFlags(commitAuthor),
		"commit", "-m", message,
	); err != nil {
		return fmt.Errorf("git: failed to commit %q (%q): %w", repo.Name, url, err)
	}
	if err := pushRepo(ctx, cwd, env); err != nil {
		return fmt.Errorf("git: failed to push %q (%q): %w", repo.Name, url, err)
	}

	return nil
}

// CreateFile implements FileCreator.
func (g *gitClient) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo Repo, content []byte, path string, message string) error {
	return g.CreateFiles(ctx, commitAuthor, repo, message, []RepoFile{{
		Path:    path,
		Content: content,
	}})
}

func keyPath(key string) (path string, cleanup func() error, err error) {
	if key == "" {
		return "", nil, pipe.Skip("private_key is empty")
	}

	path = key

	_, err = ssh.ParsePrivateKey([]byte(key))
	if isPasswordError(err) {
		return "", nil, errors.New("git: key is password-protected")
	}

	if err == nil {
		// if it can be parsed as a valid private key, we write it to a
		// temp file and use that path on GIT_SSH_COMMAND.
		f, createErr := os.CreateTemp("", "id_*")
		if createErr != nil {
			return "", nil, fmt.Errorf("git: failed to store private key: %w", createErr)
		}
		generatedPath := f.Name()
		path = generatedPath
		cleanup = func() error {
			if err := os.Remove(generatedPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("git: failed to remove private_key: %w", err)
			}
			return nil
		}
		defer func() {
			if err != nil {
				err = errors.Join(err, cleanup())
			}
		}()

		// the key needs to EOF at an empty line, seems like github actions
		// is somehow removing them.
		if !strings.HasSuffix(key, "\n") {
			key += "\n"
		}

		if _, err = io.WriteString(f, key); err != nil {
			err = errors.Join(fmt.Errorf("git: failed to store private key: %w", err), f.Close())
			path = ""
			return
		}
		if err = f.Close(); err != nil {
			err = fmt.Errorf("git: failed to store private key: %w", err)
			path = ""
			return
		}
	}

	if _, err = os.Stat(path); err != nil {
		err = fmt.Errorf("git: could not stat private_key: %w", err)
		path = ""
		return
	}

	// in any case, ensure the key has the correct permissions.
	if err = os.Chmod(path, 0o600); err != nil {
		err = fmt.Errorf("git: failed to ensure private_key permissions: %w", err)
		path = ""
		return
	}

	return path, cleanup, nil
}

func isPasswordError(err error) bool {
	var kerr *ssh.PassphraseMissingError
	return errors.As(err, &kerr)
}

func cloneRepo(ctx *context.Context, parent, url, name string, env []string) error {
	if err := retryx.Do(
		ctx,
		ctx.Config.Retry,
		func() error {
			dir := filepath.Join(parent, name)
			// Remove any leftover directory from a previous failed clone
			// attempt so that `git clone` does not fail with "already exists".
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("failed to remove partial clone directory %q: %w", dir, err)
			}
			log.WithField("url", redact.String(url, ctx.Env.Strings())).
				WithField("dir", dir).
				Info("cloning")
			return runGitCmd(ctx, parent, env, "clone", url, name)
		},
		retryx.IsNetworkError,
	); err != nil {
		return fmt.Errorf("failed to clone local repository: %w", err)
	}
	return nil
}

func pushRepo(ctx *context.Context, cwd string, env []string) error {
	return retryx.Do(
		ctx,
		ctx.Config.Retry,
		func() error {
			return runGitCmd(ctx, cwd, env, "push", "origin", "HEAD")
		},
		retryx.IsNetworkError,
	)
}

func checkoutDirName(name, url, branch string) string {
	sum := sha256.Sum256([]byte(url + "\x00" + branch))
	return fmt.Sprintf("%s-%s-%x", name, branch, sum[:8])
}

func runGitCmd(ctx *context.Context, cwd string, env []string, cmd ...string) error {
	return runGitCmdWith(ctx, cwd, env, nil, cmd...)
}

// runGitCmdWith runs a single git command in cwd. Globals are flags that git
// expects before the subcommand, e.g. -c key=value. They stay out of the error
// message.
func runGitCmdWith(ctx *context.Context, cwd string, env []string, globals []string, cmd ...string) error {
	args := append([]string{"-C", cwd}, globals...)
	args = append(args, cmd...)
	if _, err := git.Clean(git.RunWithEnv(ctx, env, args...)); err != nil {
		return fmt.Errorf("%q failed: %w", strings.Join(cmd, " "), err)
	}
	return nil
}

// commitConfigFlags returns the git flags that apply the author and its
// signing options to a single commit. Passing them per command keeps them out
// of the checkout config, which several callers share.
func commitConfigFlags(author config.CommitAuthor) []string {
	flags := []string{
		"-c", "user.name=" + author.Name,
		"-c", "user.email=" + author.Email,
		"-c", "commit.gpgSign=" + strconv.FormatBool(author.Signing.Enabled),
	}
	if !author.Signing.Enabled {
		return flags
	}
	if author.Signing.Key != "" {
		flags = append(flags, "-c", "user.signingKey="+author.Signing.Key)
	}
	if author.Signing.Program != "" {
		flags = append(flags, "-c", "gpg.program="+author.Signing.Program)
	}
	if author.Signing.Format != "" {
		flags = append(flags, "-c", "gpg.format="+author.Signing.Format)
	}
	return flags
}

func nameFromURL(url string) string {
	return strings.TrimSuffix(url[strings.LastIndex(url, "/")+1:], ".git")
}
