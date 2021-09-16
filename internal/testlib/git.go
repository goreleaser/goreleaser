package testlib

import (
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/stretchr/testify/require"
)

// GitInit inits a new git project.
func GitInit(tb testing.TB) {
	tb.Helper()
	out, err := fakeGit("init")
	require.NoError(tb, err)
	require.Contains(tb, out, "Initialized empty Git repository")
	require.NoError(tb, err)
	GitCheckoutBranch(tb, "main")
	_, _ = fakeGit("branch", "-D", "master")
}

// GitRemoteAdd adds the given url as remote.
func GitRemoteAdd(tb testing.TB, url string) {
	tb.Helper()
	out, err := fakeGit("remote", "add", "origin", url)
	require.NoError(tb, err)
	require.Empty(tb, out)
}

// GitRemoteAddWithName adds the given url as remote with given name.
func GitRemoteAddWithName(tb testing.TB, remote, url string) {
	tb.Helper()
	out, err := fakeGit("remote", "add", remote, url)
	require.NoError(tb, err)
	require.Empty(tb, out)
}

// GitCommit creates a git commits.
func GitCommit(tb testing.TB, msg string) {
	tb.Helper()
	GitCommitWithDate(tb, msg, time.Time{})
}

// GitCommitWithDate creates a git commit with a commit date.
func GitCommitWithDate(tb testing.TB, msg string, commitDate time.Time) {
	tb.Helper()
	env := (map[string]string)(nil)
	if !commitDate.IsZero() {
		env = map[string]string{
			"GIT_COMMITTER_DATE": commitDate.Format(time.RFC1123Z),
		}
	}
	out, err := fakeGitEnv(env, "commit", "--allow-empty", "-m", msg)
	require.NoError(tb, err)
	require.Contains(tb, out, "main", msg)
}

// GitTag creates a git tag.
func GitTag(tb testing.TB, tag string) {
	tb.Helper()
	out, err := fakeGit("tag", tag)
	require.NoError(tb, err)
	require.Empty(tb, out)
}

// GitBranch creates a git branch.
func GitBranch(tb testing.TB, branch string) {
	tb.Helper()
	out, err := fakeGit("branch", branch)
	require.NoError(tb, err)
	require.Empty(tb, out)
}

// GitAdd adds all files to stage.
func GitAdd(tb testing.TB) {
	tb.Helper()
	out, err := fakeGit("add", "-A")
	require.NoError(tb, err)
	require.Empty(tb, out)
}

func fakeGitEnv(env map[string]string, args ...string) (string, error) {
	allArgs := []string{
		"-c", "user.name='GoReleaser'",
		"-c", "user.email='test@goreleaser.github.com'",
		"-c", "commit.gpgSign=false",
		"-c", "log.showSignature=false",
	}
	allArgs = append(allArgs, args...)
	return git.RunEnv(env, allArgs...)
}

func fakeGit(args ...string) (string, error) {
	return fakeGitEnv(nil, args...)
}

// GitCheckoutBranch allows us to change the active branch that we're using.
func GitCheckoutBranch(tb testing.TB, name string) {
	tb.Helper()
	out, err := fakeGit("checkout", "-b", name)
	require.NoError(tb, err)
	require.Empty(tb, out)
}
