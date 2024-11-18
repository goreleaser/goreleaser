package testlib

import (
	"io"
	"testing"

	"github.com/caarlos0/log"
)

func init() {
	if !testing.Testing() {
		log.Fatal("testlib should not be used in production code!")
	}
	log.Log = log.New(io.Discard)
}
