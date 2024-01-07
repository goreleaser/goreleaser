// Package semerrgroup wraps an error group with a semaphore with configurable
// size, so you can control the number of tasks being executed simultaneously.
package semerrgroup

import (
	"sync"

	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/sync/errgroup"
)

// Group is the Semaphore ErrorGroup itself.
type Group interface {
	Go(func() error)
	Wait() error
}

// New returns a new Group of a given size.
func New(size int) Group {
	var g errgroup.Group
	g.SetLimit(size)
	return &g
}

var _ Group = &skipAwareGroup{}

// NewSkipAware returns a new Group of a given size and aware of pipe skips.
func NewSkipAware(g Group) Group {
	return &skipAwareGroup{g: g}
}

type skipAwareGroup struct {
	g       Group
	skipErr *multierror.Error
	l       sync.Mutex
}

// Go execs runs `fn` and saves the result if no error has been encountered.
func (s *skipAwareGroup) Go(fn func() error) {
	s.g.Go(func() error {
		err := fn()
		// if the err is a skip, set it for later, but return nil for now so the
		// group proceeds.
		if pipe.IsSkip(err) {
			s.l.Lock()
			defer s.l.Unlock()
			s.skipErr = multierror.Append(s.skipErr, err)
			return nil
		}
		return err
	})
}

// Wait waits for Go to complete and returns the first error encountered.
func (s *skipAwareGroup) Wait() error {
	// if we got a "real error", return it, otherwise return skipErr or nil.
	if err := s.g.Wait(); err != nil {
		return err
	}
	if s.skipErr == nil {
		return nil
	}

	if s.skipErr.Len() == 1 {
		return s.skipErr.Errors[0]
	}

	return s.skipErr
}
