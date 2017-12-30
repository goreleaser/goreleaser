package handler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHandlerOK(t *testing.T) {
	assert.NoError(t, New().Run(context.Background(), func() error {
		return nil
	}))
}

func TestHandlerErrors(t *testing.T) {
	var err = errors.New("some error")
	assert.EqualError(t, New().Run(context.Background(), func() error {
		return err
	}), err.Error())
}

func TestHandlerTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	assert.EqualError(t, New().Run(ctx, func() error {
		t.Log("slow task...")
		time.Sleep(time.Minute)
		return nil
	}), "context deadline exceeded")
}

func TestHandlerSignals(t *testing.T) {
	for _, signal := range []os.Signal{syscall.SIGINT, syscall.SIGTERM} {
		signal := signal
		t.Run(signal.String(), func(tt *testing.T) {
			tt.Parallel()
			var h = New()
			var errs = make(chan error, 1)
			go func() {
				errs <- h.Run(context.Background(), func() error {
					tt.Log("slow task...")
					time.Sleep(time.Minute)
					return nil
				})
			}()
			h.signals <- signal
			assert.EqualError(tt, <-errs, fmt.Sprintf("received: %s", signal))
		})
	}
}
