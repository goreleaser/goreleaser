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

// ErrSentinelAmbiguous is returned when the SEA fuse sentinel appears more
// than once in the binary, which would make a blind flip unsafe.
var ErrSentinelAmbiguous = errors.New("nodesea: sentinel found more than once in binary")

// ErrAlreadyFused is returned when the binary's sentinel is already in
// the "fused" state (`:1`), meaning a blob has likely already been
// injected.
var ErrAlreadyFused = errors.New("nodesea: binary already has fused sentinel")

// FlipSentinel locates the SEA fuse sentinel inside the file at path and
// flips its trailing `:0` marker to `:1`, signalling to Node.js that a
// SEA blob is attached.
//
// It returns:
//   - ErrSentinelNotFound when the sentinel cannot be located (typically
//     means the supplied binary is not a Node.js host),
//   - ErrSentinelAmbiguous when the sentinel appears more than once,
//   - ErrAlreadyFused when the sentinel is already `:1`.
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
	if bytes.Count(data, []byte(Sentinel)) > 1 {
		return ErrSentinelAmbiguous
	}

	markerAt := idx + len(Sentinel)
	if markerAt+2 > len(data) || data[markerAt] != ':' {
		return ErrSentinelNotFound
	}
	switch data[markerAt+1] {
	case '0':
		// fall through and flip
	case '1':
		return ErrAlreadyFused
	default:
		return ErrSentinelNotFound
	}

	if _, err := f.WriteAt([]byte{'1'}, int64(markerAt+1)); err != nil {
		return err
	}
	return f.Sync()
}
