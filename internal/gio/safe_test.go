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
			w.Write([]byte("a"))
			wg.Done()
		}()
	}
	wg.Wait()

	require.Len(t, b.String(), chars)
}
