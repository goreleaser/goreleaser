package nodesea

import (
	"debug/macho"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

// ErrNotSupported is returned when a binary cannot be processed because
// its sub-format is out of scope (e.g. 32-bit or universal/Fat Mach-O).
var ErrNotSupported = errors.New("nodesea: binary format not supported")

// UnsignMachO removes the LC_CODE_SIGNATURE load command and the
// trailing CMS signature blob from a 64-bit Mach-O binary at path,
// rewriting the file in place.
//
// It is conservative: the file must be a thin (non-Fat) 64-bit Mach-O,
// LC_CODE_SIGNATURE must be the last load command, and the signature
// blob must occupy the tail of the file (i.e. inside __LINKEDIT and
// reaching EOF). Anything else is rejected with ErrNotSupported — better
// to fail loudly than silently corrupt a binary.
//
// If the file has no LC_CODE_SIGNATURE the function is a no-op and
// returns nil.
func UnsignMachO(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if len(data) < 4 {
		return ErrNotSupported
	}

	// Reject Fat / 32-bit explicitly.
	magic := binary.LittleEndian.Uint32(data[:4])
	switch magic {
	case macho.Magic64:
		// ok
	case macho.Magic32, macho.MagicFat:
		return fmt.Errorf("%w: only thin 64-bit Mach-O is supported", ErrNotSupported)
	default:
		// Big-endian variants (PPC) — also out of scope.
		be := binary.BigEndian.Uint32(data[:4])
		if be == macho.MagicFat || be == macho.Magic32 || be == macho.Magic64 {
			return fmt.Errorf("%w: big-endian Mach-O is not supported", ErrNotSupported)
		}
		return fmt.Errorf("%w: not a Mach-O file (magic %#x)", ErrNotSupported, magic)
	}

	f, err := macho.NewFile(newReadSeeker(data))
	if err != nil {
		return fmt.Errorf("nodesea: parse Mach-O: %w", err)
	}

	// Locate LC_CODE_SIGNATURE.
	var (
		sigIdx                = -1
		sigOff, sigSize       uint32
		linkeditIdx           = -1
		linkeditFileoff       uint64
		linkeditFilesize      uint64
		linkeditCmdFilesizeAt int // byte offset in `data` of __LINKEDIT.filesize
		linkeditCmdOff        int // byte offset in `data` of the __LINKEDIT load cmd start
	)

	// Walk load commands ourselves to capture raw byte offsets, since
	// debug/macho hides them.
	const headerSize64 = 32
	off := headerSize64
	ncmds := int(f.Ncmd)
	for i := range ncmds {
		if off+8 > len(data) {
			return fmt.Errorf("%w: load command %d truncated", ErrNotSupported, i)
		}
		cmd := binary.LittleEndian.Uint32(data[off : off+4])
		cmdsize := binary.LittleEndian.Uint32(data[off+4 : off+8])
		if cmdsize < 8 || off+int(cmdsize) > len(data) {
			return fmt.Errorf("%w: load command %d invalid size", ErrNotSupported, i)
		}

		switch macho.LoadCmd(cmd) {
		case macho.LoadCmdSegment64:
			// segment_command_64: cmd, cmdsize, segname[16], vmaddr,
			// vmsize, fileoff, filesize, ...
			if cmdsize < 72 {
				return fmt.Errorf("%w: SEGMENT_64 too small", ErrNotSupported)
			}
			segname := cstring(data[off+8 : off+24])
			if segname == "__LINKEDIT" {
				linkeditIdx = i
				linkeditCmdOff = off
				linkeditFileoff = binary.LittleEndian.Uint64(data[off+40 : off+48])
				linkeditFilesize = binary.LittleEndian.Uint64(data[off+48 : off+56])
				linkeditCmdFilesizeAt = off + 48
			}
		case 0x1d: // LC_CODE_SIGNATURE
			if cmdsize != 16 {
				return fmt.Errorf("%w: LC_CODE_SIGNATURE wrong size (%d)", ErrNotSupported, cmdsize)
			}
			sigIdx = i
			sigOff = binary.LittleEndian.Uint32(data[off+8 : off+12])
			sigSize = binary.LittleEndian.Uint32(data[off+12 : off+16])
		}

		off += int(cmdsize)
	}

	if sigIdx < 0 {
		// Nothing to do.
		return nil
	}

	// The signature must be the last load command — that's the universal
	// invariant that signers respect. If anything else trails it we bail.
	if sigIdx != ncmds-1 {
		return fmt.Errorf("%w: LC_CODE_SIGNATURE is not the last load command", ErrNotSupported)
	}
	if linkeditIdx < 0 {
		return fmt.Errorf("%w: LC_CODE_SIGNATURE without __LINKEDIT", ErrNotSupported)
	}

	// The signature must sit at the tail of the file (i.e. last bytes of
	// __LINKEDIT). Otherwise we'd risk corrupting trailing data we don't
	// model.
	if uint64(sigOff)+uint64(sigSize) != uint64(len(data)) {
		return fmt.Errorf("%w: LC_CODE_SIGNATURE does not reach EOF", ErrNotSupported)
	}
	if uint64(sigOff) < linkeditFileoff || uint64(sigOff) >= linkeditFileoff+linkeditFilesize {
		return fmt.Errorf("%w: LC_CODE_SIGNATURE not contained in __LINKEDIT", ErrNotSupported)
	}

	// Compute new sizes.
	newFileSize := int(sigOff)
	newLinkeditFilesize := uint64(sigOff) - linkeditFileoff

	// Locate the LC_CODE_SIGNATURE load command's byte range. It is the
	// last command, so its end equals header.sizeofcmds + headerSize64.
	sizeofcmdsAt := 20 // ncmds at offset 16, sizeofcmds at offset 20
	ncmdsAt := 16
	sizeofcmds := binary.LittleEndian.Uint32(data[sizeofcmdsAt : sizeofcmdsAt+4])
	if int(sizeofcmds)+headerSize64 > len(data) {
		return fmt.Errorf("%w: header sizeofcmds out of range", ErrNotSupported)
	}
	const lcCodeSigSize = 16

	// Mutate.
	// 1. Update __LINKEDIT.filesize.
	binary.LittleEndian.PutUint64(data[linkeditCmdFilesizeAt:linkeditCmdFilesizeAt+8], newLinkeditFilesize)
	// 2. Zero the LC_CODE_SIGNATURE command bytes (within old sizeofcmds
	//    region; they fall outside the new sizeofcmds and will be ignored
	//    by dyld).
	sigCmdOff := headerSize64 + int(sizeofcmds) - lcCodeSigSize
	for i := sigCmdOff; i < sigCmdOff+lcCodeSigSize; i++ {
		data[i] = 0
	}
	// 3. Decrement ncmds and shrink sizeofcmds.
	binary.LittleEndian.PutUint32(data[ncmdsAt:ncmdsAt+4], f.Ncmd-1)
	binary.LittleEndian.PutUint32(data[sizeofcmdsAt:sizeofcmdsAt+4], sizeofcmds-lcCodeSigSize)

	// 4. Truncate (drop the signature blob).
	data = data[:newFileSize]

	// silence unused var warning: linkeditCmdOff reserved for future use
	_ = linkeditCmdOff

	// Atomic-ish replace: write to temp file then rename.
	tmp := path + ".unsign.tmp"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func cstring(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// readSeeker wraps a byte slice as an io.ReaderAt for debug/macho.
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
