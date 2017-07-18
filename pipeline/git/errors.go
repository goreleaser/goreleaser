package git

import "fmt"

// ErrInvalidVersionFormat is return when the version isnt in a valid format
type ErrInvalidVersionFormat struct {
	version string
}

func (e ErrInvalidVersionFormat) Error() string {
	return fmt.Sprintf("%v is not in a valid version format", e.version)
}

// ErrDirty happens when the repo has uncommitted/unstashed changes
type ErrDirty struct {
	status string
}

func (e ErrDirty) Error() string {
	return fmt.Sprintf("git is currently in a dirty state:\n%v", e.status)
}

// ErrWrongRef happens when the HEAD reference is different from the tag being built
type ErrWrongRef struct {
	commit, tag string
}

func (e ErrWrongRef) Error() string {
	return fmt.Sprintf("git tag %v was not made against commit %v", e.tag, e.commit)
}

// ErrNoTag happens if the underlying git repository doesn't contain any tags
// but no snapshot-release was requested.
var ErrNoTag = fmt.Errorf("git doesn't contain any tags. Either add a tag or use --snapshot")
