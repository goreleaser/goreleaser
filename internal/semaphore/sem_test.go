package semaphore

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestSemaphore(t *testing.T) {
	var sem = New(1)
	var counter = 0
	var g errgroup.Group
	for i := 0; i < 10; i++ {
		sem.Acquire()
		g.Go(func() error {
			counter++
			sem.Release()
			return nil
		})
	}
	require.NoError(t, g.Wait())
	require.Equal(t, counter, 10)
}
