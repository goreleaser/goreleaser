package logext

import (
	"bytes"
	"os"
	"strconv"
	"testing"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	t.Run("info", func(t *testing.T) {
		for _, out := range []Output{Info, Error} {
			t.Run(strconv.Itoa(int(out)), func(t *testing.T) {
				t.Cleanup(func() {
					log.Log = log.New(os.Stderr)
				})
				var b bytes.Buffer
				log.Log = log.New(&b)
				l, err := NewWriter(log.Fields{"foo": "bar"}, out).Write([]byte("foo\nbar\n"))
				require.NoError(t, err)
				require.Equal(t, 8, l)
				require.Empty(t, b.String())
			})
		}
	})

	t.Run("debug", func(t *testing.T) {
		for _, out := range []Output{Info, Error} {
			t.Run(strconv.Itoa(int(out)), func(t *testing.T) {
				t.Cleanup(func() {
					log.Log = log.New(os.Stderr)
				})
				var b bytes.Buffer
				log.Log = log.New(&b)
				log.SetLevel(log.DebugLevel)
				l, err := NewWriter(log.Fields{"foo": "bar"}, out).Write([]byte("foo\nbar\n"))
				require.NoError(t, err)
				require.Equal(t, 8, l)
				golden.RequireEqualTxt(t, b.Bytes())
			})
		}
	})
}
