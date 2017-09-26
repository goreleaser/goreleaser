package testlib

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/stretchr/testify/assert"
)

// GitInit inits a new git project
func GitInit(t *testing.T) {
	out, err := fakeGit("init")
	assert.NoError(t, err)
	assert.Contains(t, out, "Initialized empty Git repository")
	assert.NoError(t, err)
}

// GitRemoteAdd adds the given url as remote
func GitRemoteAdd(t *testing.T, url string) {
	out, err := fakeGit("remote", "add", "origin", url)
	assert.NoError(t, err)
	assert.Empty(t, out)
}

// GitCommit creates a git commits
func GitCommit(t *testing.T, msg string) {
	out, err := fakeGit("commit", "--allow-empty", "-m", msg)
	assert.NoError(t, err)
	assert.Contains(t, out, "master", msg)
}

// GitTag creates a git tag
func GitTag(t *testing.T, tag string) {
	out, err := fakeGit("tag", tag)
	assert.NoError(t, err)
	assert.Empty(t, out)
}

// GitAdd adds all files to stage
func GitAdd(t *testing.T) {
	out, err := fakeGit("add", "-A")
	assert.NoError(t, err)
	assert.Empty(t, out)
}

func fakeGit(args ...string) (string, error) {
	var allArgs = []string{
		"-c", "user.name='GoReleaser'",
		"-c", "user.email='test@goreleaser.github.com'",
		"-c", "commit.gpgSign=false",
	}
	allArgs = append(allArgs, args...)
	return git.Run(allArgs...)
}
