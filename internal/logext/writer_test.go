package logext

import (
	"os"
	"testing"

	"github.com/apex/log"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	log.SetLevel(log.DebugLevel)
	defer log.SetLevel(log.InfoLevel)
	os.Exit(m.Run())
}

func TestWriter(t *testing.T) {
	l, err := NewWriter(log.WithField("foo", "bar")).Write([]byte("foo bar\n"))
	require.NoError(t, err)
	require.Equal(t, 8, l)
}

func TestErrorWriter(t *testing.T) {
	l, err := NewErrWriter(log.WithField("foo", "bar")).Write([]byte("foo bar\n"))
	require.NoError(t, err)
	require.Equal(t, 8, l)
}
