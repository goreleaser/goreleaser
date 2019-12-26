package process

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/apex/log"
	"github.com/stretchr/testify/require"
)

func TestStream(t *testing.T) {
	var cmd = exec.Command("./testdata/script1.sh")
	require.NoError(t, Stream(cmd, NewLogWriter(log.WithField("test", "TestStream"))))
}

func TestStreamError(t *testing.T) {
	var cmd = exec.Command("./testdata/script2.sh")
	require.EqualError(t, Stream(cmd, NewLogWriter(log.WithField("test", "TestStreamError"))), "exit status 1")
}

func TestStreamWriterError(t *testing.T) {
	var cmd = exec.Command("./testdata/script1.sh")
	require.EqualError(t, Stream(cmd, ErrorWriter{}), "fake error")
}

func TestStreamCmdError(t *testing.T) {
	var cmd = exec.Command("./testdata/nope.sh")
	require.EqualError(t, Stream(cmd, ErrorWriter{}), "fork/exec ./testdata/nope.sh: no such file or directory")
}

type ErrorWriter struct{}

func (t ErrorWriter) Write(p []byte) (n int, err error) { return 0, fmt.Errorf("fake error") }
