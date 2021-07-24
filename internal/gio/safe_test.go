package gio

import (
	"bytes"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSafe(t *testing.T) {
	chars := 30

	var b bytes.Buffer
	w := Safe(&b)

	var wg sync.WaitGroup
	wg.Add(chars)
	for i := 0; i < chars; i++ {
		go func() {
			s, err := w.Write([]byte("a"))
			require.Equal(t, 1, s)
			require.NoError(t, err)
			wg.Done()
		}()
	}
	wg.Wait()

	require.Len(t, b.String(), chars)
}
