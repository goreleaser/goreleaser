// Package process provides utilities to deal with process execs.
package process

import (
	"bufio"
	"io"
	"os/exec"

	"github.com/apex/log"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// Stream streams the output of the command and waits for it to finish
func Stream(cmd *exec.Cmd, out io.Writer) error {
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	var wg errgroup.Group
	wg.Go(func() error {
		return printReader(stdout, out)
	})
	wg.Go(func() error {
		return printReader(stderr, out)
	})

	if err := wg.Wait(); err != nil {
		return err
	}
	return cmd.Wait()
}

func printReader(rd io.Reader, out io.Writer) error {
	r := bufio.NewReader(rd)
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrap(err, "failed to read line")
		}
		if _, err := out.Write(line); err != nil {
			return err
		}
	}
	return nil
}

// LogWriter writes with log.Info
type LogWriter struct {
	ctx *log.Entry
}

// NewLogWriter creates a new log writer
func NewLogWriter(ctx *log.Entry) LogWriter {
	return LogWriter{ctx: ctx}
}

func (t LogWriter) Write(p []byte) (n int, err error) {
	t.ctx.Info(string(p))
	return len(p), nil
}
