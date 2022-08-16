package testlib

import (
	"io"

	"github.com/caarlos0/log"
)

func init() {
	log.Log = log.New(io.Discard)
}
