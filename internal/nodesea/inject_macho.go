package nodesea

import (
	"debug/macho"
	"encoding/binary"
	"fmt"
	"os"
)

// MachOSegmentName is the segment name we wrap our blob section in. Node
// looks the section up by (segment, section) name pair via getsectdata,
// so changing this would break the runtime.
const MachOSegmentName = "NODE_SEA"

// MachOSectionName is the Mach-O section name expected by Node's SEA
// loader.
const MachOSectionName = "NODE_SEA_BLOB"

// InjectMachO injects blob into a 64-bit Mach-O at path as a new
// `MachOSegmentName/MachOSectionName` section, then flips the SEA fuse
// sentinel.
//
// Limitations: thin Mach-O 64-bit only. The host must have enough free
// bytes between the end of the load-command region and the first section
// payload to accommodate the new segment+section load command; this is
// always the case for stock Node builds.
func InjectMachO(path string, blob []byte) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if len(data) < 4 {
		return ErrNotSupported
	}
	magic := binary.LittleEndian.Uint32(data[:4])
	if magic != macho.Magic64 {
		return fmt.Errorf("%w: only thin 64-bit Mach-O is supported", ErrNotSupported)
	}

	f, err := macho.NewFile(newReadSeeker(data))
	if err != nil {
		return fmt.Errorf("nodesea: parse Mach-O: %w", err)
	}
	defer f.Close()

	// Idempotency: refuse to re-inject.
	if seg := f.Segment(MachOSegmentName); seg != nil {
		return ErrAlreadyInjected
	}

	const (
		hdrSize    = 32
		seg64Size  = 72
		sect64Size = 80
		lcSize     = seg64Size + sect64Size

		ncmdsAt      = 16
		sizeofcmdsAt = 20
	)

	ncmds := binary.LittleEndian.Uint32(data[ncmdsAt : ncmdsAt+4])
	sizeofcmds := binary.LittleEndian.Uint32(data[sizeofcmdsAt : sizeofcmdsAt+4])
	lcEnd := hdrSize + int(sizeofcmds)

	// Compute headroom between LC end and first section payload.
	firstSectOff := uint64(len(data))
	maxVMEnd := uint64(0)
	for _, seg := range f.Loads {
		s, ok := seg.(*macho.Segment)
		if !ok {
			continue
		}
		if s.Name == "__PAGEZERO" {
			// vmsize huge but no file backing; ignore for headroom.
			if end := s.Addr + s.Memsz; end > maxVMEnd {
				maxVMEnd = end
			}
			continue
		}
		if s.Filesz > 0 && s.Offset < firstSectOff {
			firstSectOff = s.Offset
		}
		if end := s.Addr + s.Memsz; end > maxVMEnd {
			maxVMEnd = end
		}
	}
	if firstSectOff < uint64(lcEnd) {
		return fmt.Errorf("%w: load commands overlap section data", ErrNotSupported)
	}
	if int(firstSectOff)-lcEnd < lcSize {
		return fmt.Errorf("%w: not enough room for a new load command (need %d bytes, have %d)",
			ErrNotSupported, lcSize, int(firstSectOff)-lcEnd)
	}

	// Page-align new segment vmaddr (4K page).
	const pageSize = 0x4000 // 16K, conservative for arm64
	newVmaddr := (maxVMEnd + pageSize - 1) &^ (pageSize - 1)
	newFileoff := uint64(len(data))
	// Pad blob to page boundary for clean fileoff alignment.
	blobLen := uint64(len(blob))
	vmsize := (blobLen + pageSize - 1) &^ (pageSize - 1)

	// Build the new SEGMENT_64 + section_64 LC.
	lcBuf := make([]byte, lcSize)
	binary.LittleEndian.PutUint32(lcBuf[0:], uint32(macho.LoadCmdSegment64))
	binary.LittleEndian.PutUint32(lcBuf[4:], uint32(lcSize))
	copyName(lcBuf[8:24], MachOSegmentName)
	binary.LittleEndian.PutUint64(lcBuf[24:], newVmaddr) // vmaddr
	binary.LittleEndian.PutUint64(lcBuf[32:], vmsize)    // vmsize
	binary.LittleEndian.PutUint64(lcBuf[40:], newFileoff)
	binary.LittleEndian.PutUint64(lcBuf[48:], blobLen)
	binary.LittleEndian.PutUint32(lcBuf[56:], 1) // maxprot = VM_PROT_READ
	binary.LittleEndian.PutUint32(lcBuf[60:], 1) // initprot = VM_PROT_READ
	binary.LittleEndian.PutUint32(lcBuf[64:], 1) // nsects
	binary.LittleEndian.PutUint32(lcBuf[68:], 0) // flags

	// section_64.
	sect := lcBuf[seg64Size:]
	copyName(sect[0:16], MachOSectionName)
	copyName(sect[16:32], MachOSegmentName)
	binary.LittleEndian.PutUint64(sect[32:], newVmaddr)          // addr
	binary.LittleEndian.PutUint64(sect[40:], blobLen)            // size
	binary.LittleEndian.PutUint32(sect[48:], uint32(newFileoff)) // offset
	binary.LittleEndian.PutUint32(sect[52:], 0)                  // align (2^0 = 1)
	binary.LittleEndian.PutUint32(sect[56:], 0)                  // reloff
	binary.LittleEndian.PutUint32(sect[60:], 0)                  // nreloc
	binary.LittleEndian.PutUint32(sect[64:], 0)                  // flags
	binary.LittleEndian.PutUint32(sect[68:], 0)                  // reserved1
	binary.LittleEndian.PutUint32(sect[72:], 0)                  // reserved2
	binary.LittleEndian.PutUint32(sect[76:], 0)                  // reserved3

	// Mutate.
	out := make([]byte, len(data))
	copy(out, data)
	// Insert LC at lcEnd.
	copy(out[lcEnd:lcEnd+lcSize], lcBuf)
	// Update header.
	binary.LittleEndian.PutUint32(out[ncmdsAt:], ncmds+1)
	binary.LittleEndian.PutUint32(out[sizeofcmdsAt:], sizeofcmds+lcSize)
	// Append blob at EOF.
	out = append(out, blob...)

	tmp := path + ".inject.tmp"
	if err := os.WriteFile(tmp, out, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return FlipSentinel(path)
}

func copyName(dst []byte, name string) {
	for i := range dst {
		dst[i] = 0
	}
	copy(dst, name)
}
