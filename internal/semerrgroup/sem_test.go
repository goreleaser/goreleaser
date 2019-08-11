package semerrgroup

import (
	"fmt"
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
	var num = 10
	var g = New(1)
	var output = []int{}
	for i := 0; i < num; i++ {
		i := i
		g.Go(func() error {
			output = append(output, i)
			return nil
		})
	}
	require.NoError(t, g.Wait())
	require.Equal(t, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, output)
}

func TestSemaphoreOrderError(t *testing.T) {
	var g = New(1)
	var output = []int{}
	for i := 0; i < 10; i++ {
		i := i
		g.Go(func() error {
			output = append(output, i)
			return fmt.Errorf("fake err")
		})
	}
	require.EqualError(t, g.Wait(), "fake err")
	require.Equal(t, []int{0}, output)
}
