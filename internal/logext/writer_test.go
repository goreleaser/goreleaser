package logext

import (
	"testing"

	"github.com/apex/log"
	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	l, err := NewWriter(log.WithField("foo", "bar")).Write([]byte("foo bar"))
	require.NoError(t, err)
	require.Equal(t, 7, l)
}
