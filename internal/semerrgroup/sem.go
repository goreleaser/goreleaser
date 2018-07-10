// Package semerrgroup wraps a error group with a semaphore with configurable
// size, so you can control the number of tasks being executed simultaneously.
package semerrgroup

import "golang.org/x/sync/errgroup"

// Group is the Semphore ErrorGroup itself
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
	s.g.Go(func() error {
		s.ch <- true
		defer func() {
			<-s.ch
		}()
		return fn()
	})
}

// Wait waits for the group to complete and return an error if any.
func (s *Group) Wait() error {
	return s.g.Wait()
}
