package testlib

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

// GitInit inits a new git project
func GitInit(t *testing.T) {
	var assert = assert.New(t)
	out, err := git("init")
	assert.NoError(err)
	assert.Contains(out, "Initialized empty Git repository")
	assert.NoError(err)
}

// GitRemoteAdd adds the given url as remote
func GitRemoteAdd(t *testing.T, url string) {
	var assert = assert.New(t)
	out, err := fakeGit("remote", "add", "origin", url)
	assert.NoError(err)
	assert.Empty(out)
}

// GitCommit creates a git commits
func GitCommit(t *testing.T, msg string) {
	var assert = assert.New(t)
	out, err := fakeGit("commit", "--allow-empty", "-m", msg)
	assert.NoError(err)
	assert.Contains(out, "master", msg)
}

// GitTag creates a git tag
func GitTag(t *testing.T, tag string) {
	var assert = assert.New(t)
	out, err := fakeGit("tag", tag)
	assert.NoError(err)
	assert.Empty(out)
}

// GitAdd adds all files to stage
func GitAdd(t *testing.T) {
	var assert = assert.New(t)
	out, err := git("add", "-A")
	assert.NoError(err)
	assert.Empty(out)
}

func fakeGit(args ...string) (string, error) {
	var allArgs = []string{
		"-c", "user.name='GoReleaser'",
		"-c", "user.email='test@goreleaser.github.com'",
		"-c", "commit.gpgSign=false",
	}
	allArgs = append(allArgs, args...)
	return git(allArgs...)
}

func git(args ...string) (output string, err error) {
	var cmd = exec.Command("git", args...)
	bts, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.New(string(bts))
	}
	return string(bts), err
}
