package logext

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/golden"
)

func TestDuration(t *testing.T) {
	t.Setenv("CI", "")
	t.Cleanup(func() {
		log.Log = log.New(os.Stderr)
	})
	var b bytes.Buffer
	log.Log = log.New(&b)
	log.Info("before")
	Duration(time.Now().Add(-10*time.Second), time.Minute)
	Duration(time.Now().Add(-10*time.Minute), time.Minute)
	log.Info("after")
	golden.RequireEqualTxt(t, b.Bytes())
}
