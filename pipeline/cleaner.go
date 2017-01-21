package pipeline

import "github.com/goreleaser/goreleaser/context"

// Cleaner is an interface that a pipe can implement
// to cleanup after all pipes ran.
type Cleaner interface {
	// Clean is called after pipeline is done - even if a pipe returned an error.
	Clean(*context.Context)
}
