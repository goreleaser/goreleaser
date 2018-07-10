// Package semerrgroup provides a small and simple semaphore lib for goreleaser.
package semerrgroup

import "golang.org/x/sync/errgroup"

// Group is the Group itself
type Group struct {
	ch chan bool
	g  errgroup.Group
}

// New returns a new Group of a given size.
func New(size int) *Group {
	return &Group{
		ch: make(chan bool, size),
		g:  errgroup.Group{},
	}
}

// Go execs one function respecting the group and semaphore.
func (s *Group) Go(fn func() error) {
	s.ch <- true
	s.g.Go(func() error {
		defer func() {
			<-s.ch
		}()
		return fn()
	})
}

// Release releases one Group permit
func (s *Group) Wait() error {
	return s.g.Wait()
}
