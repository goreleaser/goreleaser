package logext

import (
	"io"

	"github.com/apex/log"
)

// a io.Writer writes with log.Info.
type infoWriter struct {
	ctx *log.Entry
}

// NewWriter creates a new log writer.
func NewWriter(ctx *log.Entry) io.Writer {
	if isDebug() {
		return infoWriter{ctx: ctx}
	}
	return io.Discard
}

func (t infoWriter) Write(p []byte) (n int, err error) {
	t.ctx.Info(string(p))
	return len(p), nil
}

// a io.Writer tha writes with log.Error.
type errorWriter struct {
	ctx *log.Entry
}

// NewErrWriter creates a new log writer.
func NewErrWriter(ctx *log.Entry) io.Writer {
	if isDebug() {
		return errorWriter{ctx: ctx}
	}
	return io.Discard
}

func (w errorWriter) Write(p []byte) (n int, err error) {
	w.ctx.Error(string(p))
	return len(p), nil
}

func isDebug() bool {
	logger, ok := log.Log.(*log.Logger)
	return ok && logger.Level == log.DebugLevel
}
