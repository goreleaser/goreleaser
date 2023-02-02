package logext

import (
	"io"
	"os"

	"github.com/caarlos0/log"
)

// NewWriter creates a new log writer.
func NewWriter() io.Writer {
	return NewConditionalWriter(false)
}

// NewConditionalWriter creates a new log writer that only writes when the given condition is met or debug is enabled.
func NewConditionalWriter(condition bool) io.Writer {
	if condition || isDebug() {
		logger, ok := log.Log.(*log.Logger)
		if !ok {
			return os.Stderr
		}
		return logger.Writer
	}
	return io.Discard
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
