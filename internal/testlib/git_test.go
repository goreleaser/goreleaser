package testlib

import (
	"testing"
)

func TestGit(t *testing.T) {
	TestMkTemp(t)
	GitInit(t)
	GitAdd(t)
	GitCommit(t, "commit1")
	GitRemoteAdd(t, "git@github.com:goreleaser/nope.git")
	GitTag(t, "v1.0.0")
}
