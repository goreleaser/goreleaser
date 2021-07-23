package logext

import (
	"bytes"
	"os"
	"strconv"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	t.Run("info", func(t *testing.T) {
		for _, out := range []Output{Info, Error} {
			t.Run(strconv.Itoa(int(out)), func(t *testing.T) {
				t.Cleanup(func() {
					cli.Default.Writer = os.Stderr
				})
				var b bytes.Buffer
				cli.Default.Writer = &b
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
					cli.Default.Writer = os.Stderr
					log.SetLevel(log.InfoLevel)
				})
				log.SetLevel(log.DebugLevel)
				var b bytes.Buffer
				cli.Default.Writer = &b
				l, err := NewWriter(log.Fields{"foo": "bar"}, out).Write([]byte("foo\nbar\n"))
				require.NoError(t, err)
				require.Equal(t, 8, l)
				golden.RequireEqualTxt(t, b.Bytes())
			})
		}
	})
}
