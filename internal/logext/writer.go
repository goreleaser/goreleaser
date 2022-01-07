package logext

import (
	"bytes"
	"io"
	"strings"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
)

// Output type of the log output.
type Output int

const (
	// Info usually is used with stdout.
	Info Output = iota

	// Error usually is used with stderr.
	Error
)

// NewWriter creates a new log writer.
func NewWriter(fields log.Fields, out Output) io.Writer {
	return NewConditionalWriter(fields, out, false)
}

// NewConditionalWriter creates a new log writer that only writes when the given condition is met or debug is enabled.
func NewConditionalWriter(fields log.Fields, out Output, condition bool) io.Writer {
	if condition || isDebug() {
		return logWriter{
			ctx: newLogger(fields),
			out: out,
		}
	}
	return io.Discard
}

type logWriter struct {
	ctx *log.Entry
	out Output
}

func (w logWriter) Write(p []byte) (int, error) {
	for _, line := range strings.Split(toString(p), "\n") {
		switch w.out {
		case Info:
			w.ctx.Info(line)
		case Error:
			w.ctx.Warn(line)
		}
	}
	return len(p), nil
}

func newLogger(fields log.Fields) *log.Entry {
	handler := cli.New(cli.Default.Writer)
	handler.Padding = cli.Default.Padding + 3
	return (&log.Logger{
		Handler: handler,
		Level:   log.InfoLevel,
	}).WithFields(fields)
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
