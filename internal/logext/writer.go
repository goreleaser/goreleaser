package logext

import "github.com/apex/log"

// Writer writes with log.Info.
type Writer struct {
	ctx *log.Entry
}

// NewWriter creates a new log writer.
func NewWriter(ctx *log.Entry) Writer {
	return Writer{ctx: ctx}
}

func (t Writer) Write(p []byte) (n int, err error) {
	t.ctx.Info(string(p))
	return len(p), nil
}

// Writer writes with log.Error.
type ErrorWriter struct {
	ctx *log.Entry
}

// NewWriter creates a new log writer.
func NewErrWriter(ctx *log.Entry) ErrorWriter {
	return ErrorWriter{ctx: ctx}
}

func (w ErrorWriter) Write(p []byte) (n int, err error) {
	w.ctx.Error(string(p))
	return len(p), nil
}
