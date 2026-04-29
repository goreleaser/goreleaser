package nodesea

import (
	"bytes"
	"errors"
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

// flipSentinel returns data with the SEA fuse sentinel's trailing
// `:0` marker flipped to `:1` in place, signalling to Node.js that a
// SEA blob is attached.
//
// It returns:
//   - ErrSentinelNotFound when the sentinel cannot be located (typically
//     means the supplied binary is not a Node.js host),
//   - ErrSentinelAmbiguous when the sentinel appears more than once,
//   - ErrAlreadyFused when the sentinel is already `:1`.
func flipSentinel(data []byte) ([]byte, error) {
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
