package logext

import (
	"bytes"
	"io"
	"strings"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
)

// a io.Writer writes with log.Info.
type infoWriter struct {
	ctx *log.Entry
}

// NewWriter creates a new log writer.
func NewWriter(ctx *log.Entry) io.Writer {
	if isDebug() {
		return infoWriter{ctx: newLogger(ctx)}
	}
	return io.Discard
}

func (w infoWriter) Write(p []byte) (n int, err error) {
	for _, line := range strings.Split(toString(p), "\n") {
		w.ctx.Info(line)
	}
	return len(p), nil
}

// a io.Writer tha writes with log.Error.
type errorWriter struct {
	ctx *log.Entry
}

// NewErrWriter creates a new log writer.
func NewErrWriter(ctx *log.Entry) io.Writer {
	if isDebug() {
		return errorWriter{ctx: newLogger(ctx)}
	}
	return io.Discard
}

func (w errorWriter) Write(p []byte) (n int, err error) {
	for _, line := range strings.Split(toString(p), "\n") {
		w.ctx.Error(line)
	}
	return len(p), nil
}

func newLogger(ctx *log.Entry) *log.Entry {
	handler := cli.New(cli.Default.Writer)
	handler.Padding = cli.Default.Padding + 3
	fields := log.Fields(map[string]interface{}{})
	log := &log.Logger{
		Handler: handler,
		Level:   logLevel(),
	}
	for k, v := range ctx.Fields {
		fields[k] = v
	}
	return log.WithFields(fields)
}

func isDebug() bool {
	return logLevel() == log.DebugLevel
}

func logLevel() log.Level {
	if logger, ok := log.Log.(*log.Logger); ok {
		return logger.Level
	}
	return log.InfoLevel
}

func toString(b []byte) string {
	return string(bytes.TrimSuffix(b, []byte("\n")))
}
