package semerrgroup

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSemaphore(t *testing.T) {
	var g = New(1)
	var counter = 0
	for i := 0; i < 10; i++ {
		g.Go(func() error {
			counter++
			return nil
		})
	}
	require.NoError(t, g.Wait())
	require.Equal(t, counter, 10)
}
