package semerrgroup

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/require"
)

func TestBlockingFirst(t *testing.T) {
	g := NewBlockingFirst(New(5))
	var lock sync.Mutex
	var counter int
	for range 10 {
		g.Go(func() error {
			time.Sleep(10 * time.Millisecond)
			lock.Lock()
			defer lock.Unlock()
			counter++
			return nil
		})
	}
	require.NoError(t, g.Wait())
	require.Equal(t, 10, counter)
}

func TestBlockingFirstError(t *testing.T) {
	g := NewBlockingFirst(New(5))
	var lock sync.Mutex
	var counter int
	for range 10 {
		g.Go(func() error {
			time.Sleep(10 * time.Millisecond)
			lock.Lock()
			defer lock.Unlock()
			if counter == 0 {
				return fmt.Errorf("my error")
			}
			counter++
			return nil
		})
	}
	require.EqualError(t, g.Wait(), "my error")
	require.Equal(t, 0, counter)
}

func TestSemaphore(t *testing.T) {
	for _, i := range []int{1, 4} {
		t.Run(fmt.Sprintf("limit-%d", i), func(t *testing.T) {
			g := New(i)
			var lock sync.Mutex
			var counter int
			for range 10 {
				g.Go(func() error {
					time.Sleep(10 * time.Millisecond)
					lock.Lock()
					counter++
					lock.Unlock()
					return nil
				})
			}
			require.NoError(t, g.Wait())
			require.Equal(t, 10, counter)
		})
	}
}

func TestSemaphoreOrder(t *testing.T) {
	num := 10
	g := New(1)
	output := []int{}
	for i := range num {
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
			var lock sync.Mutex
			output := []int{}
			for i := range 10 {
				g.Go(func() error {
					lock.Lock()
					defer lock.Unlock()
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
			for range 10 {
				g.Go(func() error {
					time.Sleep(10 * time.Millisecond)
					return pipe.Skip("fake skip")
				})
			}
			merr := &multierror.Error{}
			require.ErrorAs(t, g.Wait(), &merr, "must be a multierror")
			require.Len(t, merr.Errors, 10)
		})
	}
}

func TestSemaphoreSkipAwareSingleError(t *testing.T) {
	for _, i := range []int{1, 4} {
		t.Run(fmt.Sprintf("limit-%d", i), func(t *testing.T) {
			g := NewSkipAware(New(i))
			for i := range 10 {
				g.Go(func() error {
					time.Sleep(10 * time.Millisecond)
					if i == 5 {
						return pipe.Skip("fake skip")
					}
					return nil
				})
			}
			require.EqualError(t, g.Wait(), "fake skip")
		})
	}
}

func TestSemaphoreSkipAwareNoSkips(t *testing.T) {
	for _, i := range []int{1, 4} {
		t.Run(fmt.Sprintf("limit-%d", i), func(t *testing.T) {
			g := NewSkipAware(New(i))
			for range 10 {
				g.Go(func() error {
					time.Sleep(10 * time.Millisecond)
					return nil
				})
			}
			require.NoError(t, g.Wait())
		})
	}
}

func TestSemaphoreSkipAndRealError(t *testing.T) {
	g := NewSkipAware(New(10))
	for range 100 {
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
