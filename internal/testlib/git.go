package testlib

import (
	"testing"

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
	out, err := fakeGit("commit", "--allow-empty", "-m", msg)
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

// GitAnnotatedTag creates an annotated tag.
func GitAnnotatedTag(tb testing.TB, tag, message string) {
	tb.Helper()
	out, err := fakeGit("tag", "-a", tag, "-m", message)
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

func fakeGit(args ...string) (string, error) {
	allArgs := []string{
		"-c", "user.name='GoReleaser'",
		"-c", "user.email='test@goreleaser.github.com'",
		"-c", "commit.gpgSign=false",
		"-c", "log.showSignature=false",
	}
	allArgs = append(allArgs, args...)
	return git.Run(allArgs...)
}

// GitCheckoutBranch allows us to change the active branch that we're using.
func GitCheckoutBranch(tb testing.TB, name string) {
	tb.Helper()
	out, err := fakeGit("checkout", "-b", name)
	require.NoError(tb, err)
	require.Empty(tb, out)
}
