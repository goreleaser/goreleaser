package logext

import (
	"bytes"
	"os"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	log.SetLevel(log.DebugLevel)
	defer log.SetLevel(log.InfoLevel)
	os.Exit(m.Run())
}

func TestWriter(t *testing.T) {
	t.Cleanup(func() {
		cli.Default.Writer = os.Stderr
	})
	var b bytes.Buffer
	cli.Default.Writer = &b
	l, err := NewWriter(log.WithField("foo", "bar")).Write([]byte("foo\nbar\n"))
	require.NoError(t, err)
	require.Equal(t, 8, l)
	golden.RequireEqualTxt(t, b.Bytes())
}

func TestErrorWriter(t *testing.T) {
	t.Cleanup(func() {
		cli.Default.Writer = os.Stderr
	})
	var b bytes.Buffer
	cli.Default.Writer = &b
	l, err := NewErrWriter(log.WithField("foo", "bar")).Write([]byte("foo\nbar\n"))
	require.NoError(t, err)
	require.Equal(t, 8, l)
	golden.RequireEqualTxt(t, b.Bytes())
}
