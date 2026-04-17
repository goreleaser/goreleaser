package nodesea

import (
	"bytes"
	"errors"
	"io"
	"os"
)

// ErrSentinelNotFound is returned when the SEA fuse sentinel cannot be
// located inside a candidate host binary.
var ErrSentinelNotFound = errors.New("nodesea: sentinel not found in binary")

// FlipSentinel locates the SEA fuse sentinel inside the file at path and
// sets the byte immediately following it to 1, signalling to Node.js that
// a SEA blob is attached.
//
// It returns ErrSentinelNotFound when the sentinel cannot be located,
// which typically means the supplied binary is not a Node.js host.
func FlipSentinel(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	idx := bytes.Index(data, []byte(Sentinel))
	if idx < 0 {
		return ErrSentinelNotFound
	}
	flipAt := int64(idx + len(Sentinel))
	if _, err := f.WriteAt([]byte{1}, flipAt); err != nil {
		return err
	}
	return f.Sync()
}
