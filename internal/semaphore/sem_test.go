package semaphore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSemaphore(t *testing.T) {
	var sem = New(1)
	var counter = 0
	for i := 0; i < 10; i++ {
		sem.Acquire()
		go func() {
			counter++
			sem.Release()
		}()
	}
	require.Equal(t, counter, 9)
}
