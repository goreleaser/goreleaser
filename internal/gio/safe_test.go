package gio

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafe(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		t.Helper()
		chars := 30

		var b bytes.Buffer
		w := Safe(&b)

		var wg sync.WaitGroup
		wg.Add(chars)
		for range chars {
			go func() {
				s, err := io.WriteString(w, "a")
				assert.Equal(t, 1, s)
				assert.NoError(t, err)
				wg.Done()
			}()
		}
		wg.Wait()

		require.Len(t, b.String(), chars)
	})
}
