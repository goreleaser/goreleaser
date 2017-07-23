package testlib

import (
	"testing"
)

func TestGit(t *testing.T) {
	_, back := Mktmp(t)
	defer back()
	GitInit(t)
	GitAdd(t)
	GitCommit(t, "commit1")
	GitRemoteAdd(t, "git@github.com:goreleaser/nope.git")
	GitTag(t, "v1.0.0")
}
