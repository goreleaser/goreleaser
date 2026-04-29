package nodesea

import "io"

// alignUp64 rounds v up to the nearest multiple of align (which must
// be a power of two).
func alignUp64(v, align uint64) uint64 {
	return (v + align - 1) &^ (align - 1)
}

// newReadSeeker wraps a byte slice as an io.ReaderAt so it can be passed
// to debug/macho, debug/elf and debug/pe parsers without writing to
// disk.
func newReadSeeker(b []byte) io.ReaderAt { return &byteReaderAt{b: b} }

type byteReaderAt struct{ b []byte }

func (r *byteReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || off >= int64(len(r.b)) {
		return 0, io.EOF
	}
	n := copy(p, r.b[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}
