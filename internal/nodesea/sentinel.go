package nodesea

import (
	"bytes"
	"errors"
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
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	idx, err := findFuseMarker(data)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteAt([]byte{'1'}, int64(idx)); err != nil {
		return err
	}
	return f.Sync()
}

// flipSentinelBytes returns data with the SEA fuse sentinel's trailing
// `:0` marker flipped to `:1` in place. Same semantics as
// FlipSentinel, but byte-pure so it can be chained with other
// in-memory passes.
func flipSentinelBytes(data []byte) ([]byte, error) {
	idx, err := findFuseMarker(data)
	if err != nil {
		return nil, err
	}
	data[idx] = '1'
	return data, nil
}

// findFuseMarker locates the trailing fuse byte (the `0`/`1` after
// `Sentinel:`) in data. It mirrors postject's runtime contract:
// exactly one sentinel, in `:0` state. Returns the absolute index of
// the marker byte.
func findFuseMarker(data []byte) (int, error) {
	idx := bytes.Index(data, []byte(Sentinel))
	if idx < 0 {
		return 0, ErrSentinelNotFound
	}
	if bytes.Count(data, []byte(Sentinel)) > 1 {
		return 0, ErrSentinelAmbiguous
	}
	markerAt := idx + len(Sentinel)
	if markerAt+2 > len(data) || data[markerAt] != ':' {
		return 0, ErrSentinelNotFound
	}
	switch data[markerAt+1] {
	case '0':
		return markerAt + 1, nil
	case '1':
		return 0, ErrAlreadyFused
	default:
		return 0, ErrSentinelNotFound
	}
}
