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
// loader. Note the leading "__": postject's API auto-prepends "__" to
// any section name not already starting with it, so the actual section
// name in the binary must include those two underscores.
const MachOSectionName = "__NODE_SEA_BLOB"

// InjectMachO injects blob into a 64-bit Mach-O at path as a new
// `MachOSegmentName/MachOSectionName` section.
//
// The new segment is inserted *before* __LINKEDIT (in both file and VM
// space), with __LINKEDIT shifted forward by the padded blob size. This
// keeps __LINKEDIT as the last segment in the file, which is required by
// macOS codesign for the user to be able to (re-)sign the resulting
// binary later via the goreleaser `signs` pipe.
//
// Limitations: thin Mach-O 64-bit only. The host must have enough free
// bytes between the end of the load-command region and the first section
// payload to accommodate the new segment+section load command; this is
// always the case for stock Node builds (after [UnsignMachO] strips the
// optional load commands to free up space).
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

		// 16K page is conservative across both x86_64 and arm64; the
		// actual segalign on stock Node Mach-O binaries is 0x4000.
		pageSize = 0x4000
	)

	ncmds := binary.LittleEndian.Uint32(data[ncmdsAt : ncmdsAt+4])
	sizeofcmds := binary.LittleEndian.Uint32(data[sizeofcmdsAt : sizeofcmdsAt+4])
	lcEnd := hdrSize + int(sizeofcmds)

	// Walk LCs to locate the __LINKEDIT segment LC, the
	// LC_DYLD_CHAINED_FIXUPS LC (if any — its data references a
	// per-segment array we must grow when we add NODE_SEA), and the
	// segment-index slot at which NODE_SEA will be inserted.
	const lcDyldChainedFixups = 0x80000034
	var (
		linkeditLCOff    int
		linkeditFileoff  uint64
		linkeditVmaddr   uint64
		linkeditFilesize uint64
		linkeditVmsize   uint64
		linkeditFound    bool

		chainedLCOff   = -1
		chainedDataoff uint32
		chainedSize    uint32

		// Index of the new NODE_SEA segment in the rebuilt LC list.
		// We insert it right before __LINKEDIT, so its index is the
		// same as __LINKEDIT's pre-insert segment index.
		seaSegIdx    uint32
		seenSegments uint32
	)
	off := hdrSize
	for range ncmds {
		cmd := binary.LittleEndian.Uint32(data[off:])
		sz := int(binary.LittleEndian.Uint32(data[off+4:]))
		switch cmd {
		case uint32(macho.LoadCmdSegment64):
			name := cstr(data[off+8 : off+24])
			if name == "__LINKEDIT" {
				linkeditLCOff = off
				linkeditVmaddr = binary.LittleEndian.Uint64(data[off+24:])
				linkeditVmsize = binary.LittleEndian.Uint64(data[off+32:])
				linkeditFileoff = binary.LittleEndian.Uint64(data[off+40:])
				linkeditFilesize = binary.LittleEndian.Uint64(data[off+48:])
				linkeditFound = true
				seaSegIdx = seenSegments
			}
			seenSegments++
		case lcDyldChainedFixups:
			chainedLCOff = off
			chainedDataoff = binary.LittleEndian.Uint32(data[off+8:])
			chainedSize = binary.LittleEndian.Uint32(data[off+12:])
		}
		off += sz
	}
	if !linkeditFound {
		return fmt.Errorf("%w: __LINKEDIT segment not found", ErrNotSupported)
	}
	if linkeditFileoff+linkeditFilesize != uint64(len(data)) {
		return fmt.Errorf("%w: __LINKEDIT does not end at EOF (was the binary modified?)", ErrNotSupported)
	}

	// Headroom check: we need lcSize free bytes between the end of
	// the LC region and the first section's file payload (or first
	// non-LC segment payload, whichever is earlier).
	firstSectOff := uint64(len(data))
	for _, sect := range f.Sections {
		if sect.Offset == 0 || sect.Size == 0 {
			continue
		}
		if uint64(sect.Offset) < firstSectOff {
			firstSectOff = uint64(sect.Offset)
		}
	}
	for _, l := range f.Loads {
		seg, ok := l.(*macho.Segment)
		if !ok {
			continue
		}
		if seg.Filesz == 0 {
			continue
		}
		// Skip the segment that contains the LCs themselves (its
		// fileoff is 0 and its data overlaps with the header/LC
		// region).
		if seg.Offset == 0 {
			continue
		}
		if seg.Offset < firstSectOff {
			firstSectOff = seg.Offset
		}
	}
	if firstSectOff < uint64(lcEnd) {
		return fmt.Errorf("%w: load commands overlap section data", ErrNotSupported)
	}
	if int(firstSectOff)-lcEnd < lcSize {
		return fmt.Errorf("%w: not enough room for a new load command (need %d bytes, have %d)",
			ErrNotSupported, lcSize, int(firstSectOff)-lcEnd)
	}

	blobLen := uint64(len(blob))

	// If the host has a chained-fixups blob, it embeds a per-segment
	// starts table whose seg_count must match the actual number of
	// SEGMENT_64 LCs. Adding NODE_SEA invalidates that table, so we
	// build a patched copy of the blob and append it to the END of
	// __LINKEDIT (where it stays — chained fixups data must live
	// inside __LINKEDIT for dyld to accept it). The original blob
	// bytes are left in place but become unreferenced.
	const fixupsAlign = 8
	var (
		newFixups       []byte
		newFixupsPadded uint64
	)
	if chainedLCOff >= 0 {
		patched, err := patchChainedFixupsForNewSegment(
			data[chainedDataoff:chainedDataoff+chainedSize], seaSegIdx)
		if err != nil {
			return fmt.Errorf("nodesea: patch chained fixups: %w", err)
		}
		newFixups = patched
		newFixupsPadded = (uint64(len(patched)) + fixupsAlign - 1) &^ (fixupsAlign - 1)
	}

	// delta = padded size of the new NODE_SEA segment.
	delta := (blobLen + pageSize - 1) &^ (pageSize - 1)

	// linkeditExtra = bytes appended to __LINKEDIT for the patched
	// fixups blob. We deliberately keep this equal to newFixupsPadded
	// (i.e. only 8-byte aligned, not page-aligned) so we don't write
	// any trailing zero padding past the end of __LINKEDIT — codesign's
	// strict validator rejects binaries with file bytes after the last
	// segment.
	linkeditExtra := newFixupsPadded

	// NODE_SEA takes __LINKEDIT's old VM/file slot; __LINKEDIT slides
	// forward by delta.
	newSegVmaddr := linkeditVmaddr
	newSegFileoff := linkeditFileoff

	// Build the new SEGMENT_64 + section_64 LC. We deliberately set
	// segment.filesize to delta (the page-aligned size) rather than
	// blobLen, so the file region between the blob and the next
	// segment is fully covered by NODE_SEA. This is required by
	// codesign's strict validation, which rejects binaries with
	// unmapped file regions between segments.
	lcBuf := make([]byte, lcSize)
	binary.LittleEndian.PutUint32(lcBuf[0:], uint32(macho.LoadCmdSegment64))
	binary.LittleEndian.PutUint32(lcBuf[4:], uint32(lcSize))
	copyName(lcBuf[8:24], MachOSegmentName)
	binary.LittleEndian.PutUint64(lcBuf[24:], newSegVmaddr)  // vmaddr
	binary.LittleEndian.PutUint64(lcBuf[32:], delta)         // vmsize
	binary.LittleEndian.PutUint64(lcBuf[40:], newSegFileoff) // fileoff
	binary.LittleEndian.PutUint64(lcBuf[48:], delta)         // filesize
	binary.LittleEndian.PutUint32(lcBuf[56:], 1)             // maxprot = VM_PROT_READ
	binary.LittleEndian.PutUint32(lcBuf[60:], 1)             // initprot = VM_PROT_READ
	binary.LittleEndian.PutUint32(lcBuf[64:], 1)             // nsects
	binary.LittleEndian.PutUint32(lcBuf[68:], 0)             // flags

	// section_64.
	sect := lcBuf[seg64Size:]
	copyName(sect[0:16], MachOSectionName)
	copyName(sect[16:32], MachOSegmentName)
	binary.LittleEndian.PutUint64(sect[32:], newSegVmaddr)          // addr
	binary.LittleEndian.PutUint64(sect[40:], blobLen)               // size
	binary.LittleEndian.PutUint32(sect[48:], uint32(newSegFileoff)) // offset
	binary.LittleEndian.PutUint32(sect[52:], 0)                     // align (2^0 = 1)
	binary.LittleEndian.PutUint32(sect[56:], 0)                     // reloff
	binary.LittleEndian.PutUint32(sect[60:], 0)                     // nreloc
	binary.LittleEndian.PutUint32(sect[64:], 0)                     // flags
	binary.LittleEndian.PutUint32(sect[68:], 0)                     // reserved1
	binary.LittleEndian.PutUint32(sect[72:], 0)                     // reserved2
	binary.LittleEndian.PutUint32(sect[76:], 0)                     // reserved3

	// Build the new LC region IN PLACE within the existing slack
	// between lcEnd and firstSectOff. We keep all LCs in their
	// original order, but insert the NODE_SEA LC immediately before
	// the __LINKEDIT LC. We must NOT shift any bytes past firstSectOff
	// — section data file offsets in the section headers stay valid.
	totalGrowth := delta + linkeditExtra
	out := make([]byte, len(data)+int(totalGrowth))
	copy(out, data[:linkeditFileoff])

	// Rewrite LC region.
	newLCRegion := make([]byte, 0, int(sizeofcmds)+lcSize)
	newLCRegion = append(newLCRegion, data[hdrSize:linkeditLCOff]...)
	newLCRegion = append(newLCRegion, lcBuf...)
	newLCRegion = append(newLCRegion, data[linkeditLCOff:lcEnd]...)
	if hdrSize+len(newLCRegion) > int(firstSectOff) {
		return fmt.Errorf("%w: rebuilt LC region exceeds first section offset", ErrNotSupported)
	}
	copy(out[hdrSize:], newLCRegion)
	for i := hdrSize + len(newLCRegion); i < int(firstSectOff); i++ {
		out[i] = 0
	}

	// Update __LINKEDIT LC (now at original linkeditLCOff + lcSize).
	leAt := linkeditLCOff + lcSize
	binary.LittleEndian.PutUint64(out[leAt+24:], linkeditVmaddr+delta)
	binary.LittleEndian.PutUint64(out[leAt+32:], linkeditVmsize+linkeditExtra)
	binary.LittleEndian.PutUint64(out[leAt+40:], linkeditFileoff+delta)
	binary.LittleEndian.PutUint64(out[leAt+48:], linkeditFilesize+newFixupsPadded)

	// Update Mach-O header: ncmds += 1, sizeofcmds += lcSize.
	binary.LittleEndian.PutUint32(out[ncmdsAt:], ncmds+1)
	binary.LittleEndian.PutUint32(out[sizeofcmdsAt:], sizeofcmds+uint32(lcSize))

	// Walk the new LC region and shift every linkedit_data-style file
	// offset forward by delta.
	if err := shiftLinkeditFileOffsets(out, ncmds+1, linkeditFileoff, delta); err != nil {
		return err
	}

	// Write the SEA blob into the new NODE_SEA file region.
	copy(out[linkeditFileoff:], blob)
	// Zero-padded gap to delta is already zero-init.

	// Write the shifted __LINKEDIT data.
	copy(out[linkeditFileoff+delta:], data[linkeditFileoff:])

	// Append patched chained-fixups blob at the END of __LINKEDIT and
	// redirect LC_DYLD_CHAINED_FIXUPS to it.
	if newFixups != nil {
		newFixupsFileoff := linkeditFileoff + delta + linkeditFilesize
		copy(out[newFixupsFileoff:], newFixups)
		newChainedLCOff := chainedLCOff
		if chainedLCOff >= linkeditLCOff {
			newChainedLCOff += lcSize
		}
		binary.LittleEndian.PutUint32(out[newChainedLCOff+8:], uint32(newFixupsFileoff))
		binary.LittleEndian.PutUint32(out[newChainedLCOff+12:], uint32(len(newFixups)))
	}

	tmp := path + ".inject.tmp"
	if err := os.WriteFile(tmp, out, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return FlipSentinel(path)
}

// shiftLinkeditFileOffsets walks the LC region of buf and adjusts every
// LC field that holds an absolute file offset into __LINKEDIT, adding
// delta if the original offset is at or past oldLEStart.
func shiftLinkeditFileOffsets(buf []byte, ncmds uint32, oldLEStart, delta uint64) error {
	const hdrSize = 32
	off := hdrSize
	shift := func(at int) {
		v := uint64(binary.LittleEndian.Uint32(buf[at:]))
		if v >= oldLEStart && v != 0 {
			binary.LittleEndian.PutUint32(buf[at:], uint32(v+delta))
		}
	}
	for range ncmds {
		if off+8 > len(buf) {
			return fmt.Errorf("%w: malformed LC region", ErrNotSupported)
		}
		cmd := binary.LittleEndian.Uint32(buf[off:])
		sz := int(binary.LittleEndian.Uint32(buf[off+4:]))
		switch cmd {
		case 0x2: // LC_SYMTAB: symoff, stroff
			shift(off + 8)
			shift(off + 16)
		case 0xb: // LC_DYSYMTAB: tocoff, modtaboff, extrefsymoff, indirectsymoff, extreloff, locreloff
			shift(off + 32)
			shift(off + 40)
			shift(off + 48)
			shift(off + 56)
			shift(off + 64)
			shift(off + 72)
		case 0x22, 0x80000022: // LC_DYLD_INFO[_ONLY]: rebase_off, bind_off, weak_bind_off, lazy_bind_off, export_off
			shift(off + 8)
			shift(off + 16)
			shift(off + 24)
			shift(off + 32)
			shift(off + 40)
		case 0x1d, // LC_CODE_SIGNATURE
			0x1e,       // LC_SEGMENT_SPLIT_INFO
			0x26,       // LC_FUNCTION_STARTS
			0x29,       // LC_DATA_IN_CODE
			0x2b,       // LC_DYLIB_CODE_SIGN_DRS
			0x2e,       // LC_LINKER_OPTIMIZATION_HINT
			0x80000033, // LC_DYLD_EXPORTS_TRIE
			0x80000034, // LC_DYLD_CHAINED_FIXUPS
			0x80000035: // LC_DYLD_CHAINED_FIXUPS variant
			// linkedit_data_command: dataoff at +8.
			shift(off + 8)
		}
		off += sz
	}
	return nil
}

func cstr(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func copyName(dst []byte, name string) {
	for i := range dst {
		dst[i] = 0
	}
	copy(dst, name)
}

// patchChainedFixupsForNewSegment returns a copy of the input
// dyld_chained_fixups blob with one additional (zero) seg_info_offset
// entry inserted at insertIdx, so that the embedded
// dyld_chained_starts_in_image table accounts for our newly-inserted
// SEGMENT_64 LC. All other internal offsets are adjusted to compensate
// for the 4-byte growth of the seg_info_offset array.
//
// Format reference: dyld's mach-o/fixup-chains.h.
func patchChainedFixupsForNewSegment(orig []byte, insertIdx uint32) ([]byte, error) {
	if len(orig) < 32 {
		return nil, fmt.Errorf("%w: chained fixups blob too small", ErrNotSupported)
	}
	startsOff := binary.LittleEndian.Uint32(orig[4:])
	impOff := binary.LittleEndian.Uint32(orig[8:])
	symOff := binary.LittleEndian.Uint32(orig[12:])
	if int(startsOff)+4 > len(orig) {
		return nil, fmt.Errorf("%w: chained fixups starts_offset out of range", ErrNotSupported)
	}
	segCount := binary.LittleEndian.Uint32(orig[startsOff:])
	if insertIdx > segCount {
		return nil, fmt.Errorf("%w: insertIdx %d > segCount %d", ErrNotSupported, insertIdx, segCount)
	}
	arrayOff := startsOff + 4
	if int(arrayOff)+int(segCount)*4 > len(orig) {
		return nil, fmt.Errorf("%w: chained fixups seg_info_offset array out of range", ErrNotSupported)
	}
	poolOff := arrayOff + segCount*4

	out := make([]byte, len(orig)+4)
	// Copy header + bytes up to and including starts_offset.
	copy(out, orig[:arrayOff])
	// New seg_count.
	binary.LittleEndian.PutUint32(out[startsOff:], segCount+1)
	// Build new seg_info_offset[]: insert 0 at insertIdx, shift all
	// non-zero entries by +4 (because the pool moves +4 within the
	// in_image struct).
	for i := uint32(0); i < segCount+1; i++ {
		var v uint32
		switch {
		case i < insertIdx:
			v = binary.LittleEndian.Uint32(orig[arrayOff+i*4:])
		case i == insertIdx:
			v = 0
		default:
			v = binary.LittleEndian.Uint32(orig[arrayOff+(i-1)*4:])
		}
		if v != 0 {
			v += 4
		}
		binary.LittleEndian.PutUint32(out[arrayOff+i*4:], v)
	}
	// Copy pool data, shifted by +4 in the file.
	copy(out[poolOff+4:], orig[poolOff:])
	// Update imports_offset / symbols_offset (relative to blob start)
	// if they live past startsOff.
	if impOff > startsOff {
		binary.LittleEndian.PutUint32(out[8:], impOff+4)
	}
	if symOff > startsOff {
		binary.LittleEndian.PutUint32(out[12:], symOff+4)
	}
	return out, nil
}
