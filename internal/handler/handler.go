// Package handler provides a way of having a task that is context-aware and
// that also deals with interrup and term signals.
// It was externalized mostly because it would be easier to test it this way.
// The name is not ideal but I couldn't think in a better one.
package handler

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// Task is function that can be executed by a Handler
type Task func() error

// Handler is the task handler
type Handler struct {
	signals chan os.Signal
	errs    chan error
}

// New returns a new handler with its internals setup.
func New() *Handler {
	return &Handler{
		signals: make(chan os.Signal, 1),
		errs:    make(chan error, 1),
	}
}

// Run executes a given task with a given context, dealing with its timeouts,
// cancels and SIGTERM and SIGINT signals.
// It will return an error if the context is canceled, if deadline exceeds,
// if a SIGTERM or SIGINT is received and of course if the task itself fails.
func (h *Handler) Run(ctx context.Context, task Task) error {
	go func() {
		if err := task(); err != nil {
			h.errs <- err
			return
		}
		h.errs <- nil
	}()
	signal.Notify(h.signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-h.errs:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case sig := <-h.signals:
		return fmt.Errorf("received: %s", sig)
	}
}
