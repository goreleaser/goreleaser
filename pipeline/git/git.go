// Package git implements the Pipe interface getting and validating the
// current git repository state
package git

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/context"
	ggit "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

// Pipe for brew deployment
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Getting and validating git state"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) (err error) {
	repo, err := ggit.PlainOpen(".")
	if err != nil {
		return
	}
	commit, tag, previous, err := getRefs(repo)
	if err != nil {
		return
	}
	if tag == nil && !ctx.Snapshot {
		return ErrNoTag
	}
	var tagName string
	if tag != nil {
		tagName = tag.Name().Short()
	}
	ctx.Git = context.GitInfo{
		CurrentTag: tagName,
		Commit:     commit.Hash().String(),
	}
	if ctx.ReleaseNotes == "" {
		diff, err := getLog(repo, commit, previous)
		if err != nil {
			return err
		}
		ctx.ReleaseNotes = fmt.Sprintf(
			"## Changelog\n\n%v",
			strings.Join(diff, "\n"),
		)
	}
	if err = setVersion(ctx); err != nil {
		return
	}
	if !ctx.Validate {
		log.Warn("skipped validations because --skip-validate is set")
		return nil
	}
	return validate(ctx, repo, tag, commit)
}

func validate(
	ctx *context.Context,
	repo *ggit.Repository,
	tag, commit *plumbing.Reference,
) (err error) {
	tree, err := repo.Worktree()
	if err != nil {
		return
	}
	status, err := tree.Status()
	if err != nil {
		return
	}
	if !status.IsClean() {
		return ErrDirty{
			status: status.String(),
		}
	}
	if ctx.Snapshot {
		return
	}
	if !regexp.MustCompile("^[0-9.]+").MatchString(ctx.Version) {
		return ErrInvalidVersionFormat{ctx.Version}
	}
	if tag.Hash().String() != commit.Hash().String() {
		return ErrWrongRef{ctx.Git}
	}
	return nil
}

func getRefs(repo *ggit.Repository) (commit, tag, previous *plumbing.Reference, err error) {
	var refs []*plumbing.Reference
	iter, err := repo.References()
	if err != nil {
		return
	}
	defer iter.Close()
	iter.ForEach(func(ref *plumbing.Reference) error {
		refs = append(refs, ref)
		return nil
	})
	reverse(refs)
	for _, ref := range refs {
		if !ref.IsTag() {
			continue
		}
		if tag == nil {
			tag = ref
		}
		if previous == nil && ref != tag {
			previous = ref
		}
	}
	commit, err = repo.Head()
	if err != nil {
		return
	}
	if previous == nil {
		previous = refs[len(refs)-1]
	}
	return
}

func getLog(
	repo *ggit.Repository,
	commit, previous *plumbing.Reference,
) (diff []string, err error) {
	iter, err := repo.Log(&ggit.LogOptions{From: commit.Hash()})
	if err != nil {
		return
	}
	defer iter.Close()
	for {
		commit, err := iter.Next()
		if err != nil {
			break
		}
		diff = append(diff, pretty(commit))
		if isParent(commit, previous) {
			break
		}
	}
	reverse(diff)
	return
}

func isParent(commit *object.Commit, previous *plumbing.Reference) bool {
	for _, parent := range commit.ParentHashes {
		if parent == previous.Hash() {
			return true
		}
	}
	return false
}

func pretty(commit *object.Commit) string {
	return fmt.Sprintf(
		"%v %v",
		commit.Hash.String(),
		strings.Split(commit.Message, "\n")[0],
	)
}

func setVersion(ctx *context.Context) (err error) {
	if ctx.Snapshot {
		snapshotName, err := getSnapshotName(ctx)
		if err != nil {
			return fmt.Errorf("failed to generate snapshot name: %s", err.Error())
		}
		ctx.Version = snapshotName
		return nil
	}
	// removes usual `v` prefix
	ctx.Version = strings.TrimPrefix(ctx.Git.CurrentTag, "v")
	return
}

func reverse(slice interface{}) {
	sort.Slice(slice, func(i, j int) bool {
		return i > j
	})
}
