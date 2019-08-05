package semerrgroup

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSemaphore(t *testing.T) {
	var g = New(4)
	var lock sync.Mutex
	var counter int
	for i := 0; i < 10; i++ {
		g.Go(func() error {
			time.Sleep(10 * time.Millisecond)
			lock.Lock()
			counter++
			lock.Unlock()
			return nil
		})
	}
	require.NoError(t, g.Wait())
	require.Equal(t, counter, 10)
}

func TestSemaphoreOrder(t *testing.T) {
	num := 10
	var g = New(1)
	output := make(chan int)
	go func() {
		for i := 0; i < num; i++ {
			require.Equal(t, <-output, i)
		}
		require.NoError(t, g.Wait())
	}()
	for i := 0; i < num; i++ {
		j := i
		g.Go(func() error {
			output <- j
			return nil
		})
	}
	require.NoError(t, g.Wait())
}
