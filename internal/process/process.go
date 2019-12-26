// Package process provides utilities to deal with process execs.
package process

import (
	"bufio"
	"fmt"
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
	defer stderr.Close()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer stdout.Close()
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

	if err := cmd.Wait(); err != nil {
		return err
	}
	return wg.Wait()
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
		if _, err := fmt.Fprintln(out, string(line)); err != nil {
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
