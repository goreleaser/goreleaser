// Package semaphore provides a small and simple semaphore lib for goreleaser.
package semaphore

// Semaphore is the semaphore itself
type Semaphore chan bool

// New returns a new semaphore of a given size.
func New(size int) Semaphore {
	return make(Semaphore, size)
}

// Acquire acquires one semaphore permit.
func (s Semaphore) Acquire() {
	s <- true
}

// Release releases one semaphore permit
func (s Semaphore) Release() {
	<-s
}
