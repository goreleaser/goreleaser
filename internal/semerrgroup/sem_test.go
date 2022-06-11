package semerrgroup

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/stretchr/testify/require"
)

func TestSemaphore(t *testing.T) {
	for _, i := range []int{1, 4} {
		t.Run(fmt.Sprintf("limit-%d", i), func(t *testing.T) {
			g := New(i)
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
		})
	}
}

func TestSemaphoreOrder(t *testing.T) {
	num := 10
	g := New(1)
	output := []int{}
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

func TestSemaphoreError(t *testing.T) {
	for _, i := range []int{1, 4} {
		t.Run(fmt.Sprintf("limit-%d", i), func(t *testing.T) {
			g := New(i)
			output := []int{}
			for i := 0; i < 10; i++ {
				i := i
				g.Go(func() error {
					output = append(output, i)
					return fmt.Errorf("fake err")
				})
			}
			require.EqualError(t, g.Wait(), "fake err")
			require.Len(t, output, 10)
		})
	}
}

func TestSemaphoreSkipAware(t *testing.T) {
	for _, i := range []int{1, 4} {
		t.Run(fmt.Sprintf("limit-%d", i), func(t *testing.T) {
			g := NewSkipAware(New(i))
			var lock sync.Mutex
			var counter int
			for i := 0; i < 10; i++ {
				g.Go(func() error {
					time.Sleep(10 * time.Millisecond)
					lock.Lock()
					counter++
					lock.Unlock()
					return pipe.Skip("fake skip")
				})
			}
			require.EqualError(t, g.Wait(), "fake skip")
			require.Equal(t, counter, 10)
		})
	}
}

func TestSemaphoreSkipAndRealError(t *testing.T) {
	g := NewSkipAware(New(20))
	for i := 0; i < 100; i++ {
		g.Go(func() error {
			time.Sleep(10 * time.Millisecond)
			return pipe.Skip("fake skip")
		})
	}
	g.Go(func() error {
		time.Sleep(10 * time.Millisecond)
		return fmt.Errorf("errrrrr")
	})
	require.EqualError(t, g.Wait(), "errrrrr")
}
