package logext

import "github.com/apex/log"

// Writer writes with log.Info
type Writer struct {
	ctx *log.Entry
}

// NewWriter creates a new log writer
func NewWriter(ctx *log.Entry) Writer {
	return Writer{ctx: ctx}
}

func (t Writer) Write(p []byte) (n int, err error) {
	t.ctx.Info(string(p))
	return len(p), nil
}
