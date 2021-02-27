package testlib

import (
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/stretchr/testify/require"
)

// GitInit inits a new git project.
func GitInit(t testing.TB) {
	out, err := fakeGit("init", "-b", "main")
	require.NoError(t, err)
	require.Contains(t, out, "Initialized empty Git repository")
	require.NoError(t, err)
}

// GitRemoteAdd adds the given url as remote.
func GitRemoteAdd(t testing.TB, url string) {
	out, err := fakeGit("remote", "add", "origin", url)
	require.NoError(t, err)
	require.Empty(t, out)
}

// GitCommit creates a git commits.
func GitCommit(t testing.TB, msg string) {
	GitCommitWithDate(t, msg, time.Time{})
}

// GitCommitWithDate creates a git commit with a commit date.
func GitCommitWithDate(t testing.TB, msg string, commitDate time.Time) {
	env := (map[string]string)(nil)
	if !commitDate.IsZero() {
		env = map[string]string{
			"GIT_COMMITTER_DATE": commitDate.Format(time.RFC1123Z),
		}
	}
	out, err := fakeGitEnv(env, "commit", "--allow-empty", "-m", msg)
	require.NoError(t, err)
	require.Contains(t, out, "main", msg)
}

// GitTag creates a git tag.
func GitTag(t testing.TB, tag string) {
	out, err := fakeGit("tag", tag)
	require.NoError(t, err)
	require.Empty(t, out)
}

// GitBranch creates a git branch.
func GitBranch(t testing.TB, branch string) {
	out, err := fakeGit("branch", branch)
	require.NoError(t, err)
	require.Empty(t, out)
}

// GitAdd adds all files to stage.
func GitAdd(t testing.TB) {
	out, err := fakeGit("add", "-A")
	require.NoError(t, err)
	require.Empty(t, out)
}

func fakeGitEnv(env map[string]string, args ...string) (string, error) {
	var allArgs = []string{
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
func GitCheckoutBranch(t testing.TB, name string) {
	out, err := fakeGit("checkout", "-b", name)
	require.NoError(t, err)
	require.Empty(t, out)
}
