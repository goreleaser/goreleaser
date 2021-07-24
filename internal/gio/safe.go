package gio

import (
	"io"
	"sync"
)

// Safe wraps the given writer to be thread-safe.
func Safe(w io.Writer) io.Writer {
	return &safeWriter{w: w}
}

type safeWriter struct {
	w io.Writer
	m sync.Mutex
}

func (s *safeWriter) Write(p []byte) (int, error) {
	s.m.Lock()
	defer s.m.Unlock()
	return s.w.Write(p)
}
