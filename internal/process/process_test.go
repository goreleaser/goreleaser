package process

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStream(t *testing.T) {
	var cmd = exec.Command("./testdata/script1.sh")
	require.NoError(t, Stream(cmd, LogWriter{}))
}

func TestStreamError(t *testing.T) {
	var cmd = exec.Command("./testdata/script2.sh")
	require.EqualError(t, Stream(cmd, LogWriter{}), "exit status 1")
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
