package artifact

import "errors"

// ErrNotChecksummable happens if you call Checksum() on an artifact type that
// doesn't support it.
// Usually safe to ignore.
var ErrNotChecksummable = errors.New("artifact type does not support checksumming")

type (
	Checksummer           func(items []*Artifact) ([]*Artifact, error)
	ChecksummingArtifacts struct {
		inner *Artifacts
	}
)

// Get evals the checksums and returns them.
func (a *ChecksummingArtifacts) Get() ([]*Artifact, error) {
	var result []*Artifact

	if list := a.inner.List(); len(list) > 0 {
		checks, err := a.inner.checksums(list)
		if err != nil {
			return nil, err
		}
		result = append(result, checks...)
	}

	return result, nil
}

// List returns both the current artifact list as well as their respective
// checksum artifacts.
func (a *ChecksummingArtifacts) List() ([]*Artifact, error) {
	checks, err := a.Get()
	if err != nil {
		return nil, err
	}

	return append(a.inner.List(), checks...), nil
}
