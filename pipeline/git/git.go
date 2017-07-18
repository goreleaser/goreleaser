// Package git implements the Pipe interface getting and validating the
// current git repository state
package git

import (
	"errors"
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
	var refs []*plumbing.Reference
	iter, err := repo.Storer.IterReferences()
	if err != nil {
		return
	}
	defer iter.Close()
	iter.ForEach(func(ref *plumbing.Reference) error {
		refs = append(refs, ref)
		return nil
	})
	sort.Slice(refs, func(i, j int) bool {
		return i > j
	})

	var commit *plumbing.Reference
	var tag *plumbing.Reference
	var previous *plumbing.Reference
	for _, ref := range refs {
		log.Info(ref.String())
		if !ref.IsTag() {
			continue
		}
		if tag == nil {
			log.Info("setting as last tag")
			tag = ref
		}
		if previous == nil && ref != tag {
			log.Info("setting as previous tag")
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
	if tag == nil && !ctx.Snapshot {
		return ErrNoTag
	}
	var tagName = ""
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
		sort.Slice(diff, func(i, j int) bool {
			return i > j
		})
		ctx.ReleaseNotes = fmt.Sprintf("## Changelog\n\n%v", strings.Join(diff, "\n"))
	}
	if err = setVersion(ctx); err != nil {
		return
	}
	if !ctx.Validate {
		log.Warn("skipped validations because --skip-validate is set")
		return nil
	}
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
	if !regexp.MustCompile("^[0-9.]+").MatchString(ctx.Version) {
		return ErrInvalidVersionFormat{ctx.Version}
	}
	log.Infof("tag: %v, commit: %v", tag, commit)
	if tag.Hash().String() != commit.Hash().String() {
		return ErrWrongRef{ctx.Git}
	}
	return nil
}

func getLog(repo *ggit.Repository, commit, previous *plumbing.Reference) (diff []string, err error) {
	citer, err := repo.Log(&ggit.LogOptions{From: commit.Hash()})
	if err != nil {
		return
	}
	citer.ForEach(func(commit *object.Commit) error {
		diff = append(
			diff,
			fmt.Sprintf(
				"%v %v",
				commit.Hash.String(),
				strings.Split(commit.Message, "\n")[0],
			),
		)
		if err := commit.Parents().ForEach(func(parent *object.Commit) error {
			if parent.Hash == previous.Hash() {
				return errors.New("break")
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	})
	return
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
