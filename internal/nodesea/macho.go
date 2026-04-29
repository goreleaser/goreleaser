package nodesea

import (
	"bytes"
	"debug/macho"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	gomacho "github.com/blacktop/go-macho"
	"github.com/blacktop/go-macho/pkg/codesign"
	cstypes "github.com/blacktop/go-macho/pkg/codesign/types"
)

// This file is the entire Mach-O SEA toolchain: it strips an existing
// signature, splices in a NODE_SEA segment carrying the SEA blob,
// flips the SEA fuse sentinel, and re-signs the result ad-hoc — all in
// a single read+sign+write cycle. It is the rewritten/fused successor
// of three earlier files (unsign_macho.go, inject_macho.go,
// codesign_macho.go) — see commit history.
//
// Why is this not "just append the blob to the file"?
//   - Node finds the blob at runtime via getsectiondata("NODE_SEA",
//     "__NODE_SEA_BLOB"), which walks load commands looking for a
//     named segment+section pair. Bytes past EOF are invisible to it,
//     so we need a real LC_SEGMENT_64.
//   - macOS codesign and the arm64 kernel require __LINKEDIT to be the
//     last segment in the file (the signature trailer lives at its
//     end). The new segment therefore has to be inserted *before*
//     __LINKEDIT, which means sliding __LINKEDIT forward and patching
//     every offset that points into it.
//   - Modern Node ships LC_DYLD_CHAINED_FIXUPS, whose payload contains
//     a per-segment-index table that becomes invalid the moment we
//     add a segment. Stripping the LC isn't safe — chained fixups are
//     dyld's load-time binding table.

// MachOSegmentName is the segment name we wrap our blob section in.
// Node looks the section up by (segment, section) name pair via
// getsectdata, so changing this would break the SEA loader at runtime.
const MachOSegmentName = "NODE_SEA"

// MachOSectionName is the Mach-O section name expected by Node's SEA
// loader. Note the leading "__": postject's API auto-prepends "__" to
// any section name not already starting with it, so the actual section
// name in the binary must include those two underscores.
const MachOSectionName = "__NODE_SEA_BLOB"

// ErrNotSupported is returned when a binary cannot be processed because
// its sub-format is out of scope (e.g. 32-bit or universal/Fat Mach-O).
var ErrNotSupported = errors.New("nodesea: binary format not supported")

// buildMachO is the full Mach-O SEA pipeline: strip any existing
// signature, splice in NODE_SEA carrying blob, flip the SEA fuse byte,
// and ad-hoc sign the result. The transformed binary is written to
// outPath atomically (sibling tempfile + rename).
//
// The ad-hoc sign step is unavoidable on Apple Silicon — the kernel
// refuses to exec unsigned arm64 binaries. id is the code-signing
// identifier; if empty, the basename of outPath (without extension) is
// used.
func buildMachO(srcPath, outPath string, blob []byte, id string) error {
	src, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	out, err := unsignMachOBytes(src)
	if err != nil {
		return fmt.Errorf("nodesea: unsign mach-o: %w", err)
	}
	out, err = injectMachOBytes(out, blob)
	if err != nil {
		return fmt.Errorf("nodesea: inject mach-o: %w", err)
	}
	out, err = flipSentinelBytes(out)
	if err != nil {
		return fmt.Errorf("nodesea: flip sentinel: %w", err)
	}

	// blacktop/go-macho's CodeSign+Save round-trips through the
	// filesystem (it has no in-memory writer), so we land in a sibling
	// tempfile, sign in place, then atomically rename to outPath.
	tmp, err := os.CreateTemp(filepath.Dir(outPath), ".nodesea-macho-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := os.WriteFile(tmpPath, out, 0o755); err != nil {
		return err
	}
	if id == "" {
		base := filepath.Base(outPath)
		id = strings.TrimSuffix(base, filepath.Ext(base))
	}
	if err := adHocSignFile(tmpPath, id); err != nil {
		return fmt.Errorf("nodesea: ad-hoc sign: %w", err)
	}
	// blacktop's f.Save unconditionally writes 0o755 — explicit chmod
	// here keeps the executable bit set across go versions even if
	// that ever changes.
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		return err
	}
	success = true
	return nil
}

// adHocSignFile applies an ad-hoc CMS signature to the Mach-O at path
// using blacktop/go-macho. id is the code-signing identifier and must
// be non-empty.
func adHocSignFile(path, id string) error {
	f, err := gomacho.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	cfg := &codesign.Config{
		ID:    id,
		Flags: cstypes.ADHOC | cstypes.LINKER_SIGNED,
	}
	if err := f.CodeSign(cfg); err != nil {
		return fmt.Errorf("sign %s: %w", path, err)
	}
	if err := f.Save(path); err != nil {
		return fmt.Errorf("save %s: %w", path, err)
	}
	return nil
}

// ---------------------------------------------------------------------
// unsign
// ---------------------------------------------------------------------

// unsignMachOBytes returns a copy of data with the LC_CODE_SIGNATURE
// load command and the trailing CMS signature blob removed. It also
// strips a small set of optional metadata load commands
// (LC_FUNCTION_STARTS, LC_DATA_IN_CODE, LC_SOURCE_VERSION) to free up
// header padding for the new SEA segment we add later — stock
// Apple-shipped Node x86_64 binaries leave only ~96 bytes of
// headerpad, which is not enough for a fresh SEGMENT_64+section_64
// load command pair (152 bytes).
//
// Stripping these LCs is safe at runtime — they are only consumed by
// debuggers/profilers (function starts), the linker for arm
// data-in-code regions, and CrashReporter (source version metadata).
//
// Conservative: thin (non-Fat) 64-bit Mach-O only, LC_CODE_SIGNATURE
// (when present) must be the last load command, and the signature blob
// must occupy the tail of the file (i.e. inside __LINKEDIT and
// reaching EOF). Anything else is rejected with ErrNotSupported.
//
// If none of the targeted LCs are present this is a copy-through.
func unsignMachOBytes(orig []byte) ([]byte, error) {
	if len(orig) < 4 {
		return nil, ErrNotSupported
	}
	switch magic := binary.LittleEndian.Uint32(orig[:4]); magic {
	case macho.Magic64:
		// ok
	case macho.Magic32, macho.MagicFat:
		return nil, fmt.Errorf("%w: only thin 64-bit Mach-O is supported", ErrNotSupported)
	default:
		be := binary.BigEndian.Uint32(orig[:4])
		if be == macho.MagicFat || be == macho.Magic32 || be == macho.Magic64 {
			return nil, fmt.Errorf("%w: big-endian Mach-O is not supported", ErrNotSupported)
		}
		return nil, fmt.Errorf("%w: not a Mach-O file (magic %#x)", ErrNotSupported, magic)
	}

	data := append([]byte(nil), orig...)
	f, err := macho.NewFile(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	const (
		hdrSize         = 32
		ncmdsAt         = 16
		sizeofcmdsAt    = 20
		lcCodeSigSize   = 16
		lcCodeSignature = 0x1d
	)
	stripExtras := map[uint32]bool{
		0x26: true, // LC_FUNCTION_STARTS
		0x29: true, // LC_DATA_IN_CODE
		0x2a: true, // LC_SOURCE_VERSION
	}

	var (
		sigIdx                                              = -1
		sigOff, sigSize                                     uint32
		linkeditIdx                                         = -1
		linkeditFileoff, linkeditFilesize                   uint64
		linkeditCmdVmsizeAt, linkeditCmdFilesizeAt          int
		extraStripped                                       []struct{ off, size int }
	)

	off := hdrSize
	ncmds := int(f.Ncmd)
	for i := range ncmds {
		if off+8 > len(data) {
			return nil, fmt.Errorf("%w: load command %d truncated", ErrNotSupported, i)
		}
		cmd := binary.LittleEndian.Uint32(data[off:])
		cmdsize := binary.LittleEndian.Uint32(data[off+4:])
		if cmdsize < 8 || off+int(cmdsize) > len(data) {
			return nil, fmt.Errorf("%w: load command %d invalid size", ErrNotSupported, i)
		}

		switch macho.LoadCmd(cmd) {
		case macho.LoadCmdSegment64:
			if cmdsize < 72 {
				return nil, fmt.Errorf("%w: SEGMENT_64 too small", ErrNotSupported)
			}
			if cstring(data[off+8:off+24]) == "__LINKEDIT" {
				linkeditIdx = i
				linkeditCmdVmsizeAt = off + 32
				linkeditCmdFilesizeAt = off + 48
				linkeditFileoff = binary.LittleEndian.Uint64(data[off+40:])
				linkeditFilesize = binary.LittleEndian.Uint64(data[off+48:])
			}
		case lcCodeSignature:
			if cmdsize != 16 {
				return nil, fmt.Errorf("%w: LC_CODE_SIGNATURE wrong size (%d)", ErrNotSupported, cmdsize)
			}
			sigIdx = i
			sigOff = binary.LittleEndian.Uint32(data[off+8:])
			sigSize = binary.LittleEndian.Uint32(data[off+12:])
		default:
			if stripExtras[cmd] {
				extraStripped = append(extraStripped, struct{ off, size int }{off, int(cmdsize)})
			}
		}
		off += int(cmdsize)
	}

	if sigIdx < 0 && len(extraStripped) == 0 {
		return data, nil
	}

	if sigIdx >= 0 {
		if sigIdx != ncmds-1 {
			return nil, fmt.Errorf("%w: LC_CODE_SIGNATURE is not the last load command", ErrNotSupported)
		}
		if linkeditIdx < 0 {
			return nil, fmt.Errorf("%w: LC_CODE_SIGNATURE without __LINKEDIT", ErrNotSupported)
		}
		if uint64(sigOff)+uint64(sigSize) != uint64(len(data)) {
			return nil, fmt.Errorf("%w: LC_CODE_SIGNATURE does not reach EOF", ErrNotSupported)
		}
		if uint64(sigOff) < linkeditFileoff || uint64(sigOff) >= linkeditFileoff+linkeditFilesize {
			return nil, fmt.Errorf("%w: LC_CODE_SIGNATURE not contained in __LINKEDIT", ErrNotSupported)
		}
	}

	sizeofcmds := binary.LittleEndian.Uint32(data[sizeofcmdsAt:])
	if int(sizeofcmds)+hdrSize > len(data) {
		return nil, fmt.Errorf("%w: header sizeofcmds out of range", ErrNotSupported)
	}

	// 1. Shrink __LINKEDIT to drop the trailing signature bytes. Both
	//    filesize AND vmsize must match — codesign --verify and Apple
	//    notarization reject binaries whose __LINKEDIT vmsize > filesize.
	if sigIdx >= 0 {
		newLinkeditSize := uint64(sigOff) - linkeditFileoff
		binary.LittleEndian.PutUint64(data[linkeditCmdFilesizeAt:], newLinkeditSize)
		binary.LittleEndian.PutUint64(data[linkeditCmdVmsizeAt:], newLinkeditSize)
	}

	// 2. Compact the load command region: drop the stripped LCs (extras
	//    anywhere; LC_CODE_SIGNATURE is last by invariant above) and
	//    shift the rest forward, preserving relative order.
	stripSet := make(map[int]int, len(extraStripped)+1)
	removedBytes := 0
	for _, s := range extraStripped {
		stripSet[s.off] = s.size
		removedBytes += s.size
	}
	removedCmds := uint32(len(extraStripped))
	if sigIdx >= 0 {
		stripSet[hdrSize+int(sizeofcmds)-lcCodeSigSize] = lcCodeSigSize
		removedBytes += lcCodeSigSize
		removedCmds++
	}

	lcEnd := hdrSize + int(sizeofcmds)
	rebuilt := make([]byte, 0, int(sizeofcmds)-removedBytes)
	cur := hdrSize
	for cur < lcEnd {
		if cur+8 > lcEnd {
			return nil, fmt.Errorf("%w: load command region truncated during compact", ErrNotSupported)
		}
		sz := int(binary.LittleEndian.Uint32(data[cur+4:]))
		if sz <= 0 || cur+sz > lcEnd {
			return nil, fmt.Errorf("%w: invalid cmdsize during compact", ErrNotSupported)
		}
		if _, drop := stripSet[cur]; !drop {
			rebuilt = append(rebuilt, data[cur:cur+sz]...)
		}
		cur += sz
	}
	if len(rebuilt) != int(sizeofcmds)-removedBytes {
		return nil, fmt.Errorf("%w: compact size mismatch (got %d want %d)",
			ErrNotSupported, len(rebuilt), int(sizeofcmds)-removedBytes)
	}
	copy(data[hdrSize:hdrSize+len(rebuilt)], rebuilt)
	for i := hdrSize + len(rebuilt); i < lcEnd; i++ {
		data[i] = 0
	}

	// 3. Decrement ncmds and shrink sizeofcmds.
	binary.LittleEndian.PutUint32(data[ncmdsAt:], f.Ncmd-removedCmds)
	binary.LittleEndian.PutUint32(data[sizeofcmdsAt:], sizeofcmds-uint32(removedBytes))

	// 4. Truncate (drop the signature blob) if applicable.
	if sigIdx >= 0 {
		data = data[:int(sigOff)]
	}
	return data, nil
}

// ---------------------------------------------------------------------
// inject
// ---------------------------------------------------------------------

// injectMachOBytes returns a new Mach-O byte slice with blob spliced in
// as a new MachOSegmentName/MachOSectionName segment+section, inserted
// before __LINKEDIT in both file and VM space. __LINKEDIT slides
// forward by the padded blob size so it stays the last segment in the
// file (mandatory for codesign to work later).
//
// Limitations: thin Mach-O 64-bit only. The host must have enough free
// bytes between the end of the load-command region and the first
// section payload to accommodate the new segment+section LC pair; this
// is always the case for stock Node builds (after unsignMachOBytes
// strips the optional load commands to free up space).
func injectMachOBytes(data, blob []byte) ([]byte, error) {
	if len(data) < 4 {
		return nil, ErrNotSupported
	}
	if binary.LittleEndian.Uint32(data[:4]) != macho.Magic64 {
		return nil, fmt.Errorf("%w: only thin 64-bit Mach-O is supported", ErrNotSupported)
	}

	f, err := macho.NewFile(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if seg := f.Segment(MachOSegmentName); seg != nil {
		return nil, ErrAlreadyInjected
	}

	const (
		hdrSize    = 32
		seg64Size  = 72
		sect64Size = 80
		lcSize     = seg64Size + sect64Size
		ncmdsAt    = 16
		sizeofcmdsAt = 20
		// 16K page is conservative across both x86_64 and arm64;
		// stock Node Mach-O binaries already use 0x4000 segalign.
		pageSize = 0x4000
		// LC_DYLD_CHAINED_FIXUPS is the modern dyld bind/rebase
		// table. Adding any segment invalidates its embedded
		// per-segment-index array, so we patch the blob.
		lcDyldChainedFixups = 0x80000034
	)

	ncmds := binary.LittleEndian.Uint32(data[ncmdsAt:])
	sizeofcmds := binary.LittleEndian.Uint32(data[sizeofcmdsAt:])
	lcEnd := hdrSize + int(sizeofcmds)

	var (
		linkeditLCOff                                                  int
		linkeditFileoff, linkeditVmaddr, linkeditFilesize, linkeditVmsize uint64
		linkeditFound                                                  bool

		chainedLCOff   = -1
		chainedDataoff uint32
		chainedSize    uint32

		// Index of the new NODE_SEA segment in the rebuilt LC list.
		// We insert it right before __LINKEDIT, so its index is the
		// pre-insert index of __LINKEDIT.
		seaSegIdx    uint32
		seenSegments uint32
	)
	off := hdrSize
	for range ncmds {
		cmd := binary.LittleEndian.Uint32(data[off:])
		sz := int(binary.LittleEndian.Uint32(data[off+4:]))
		switch cmd {
		case uint32(macho.LoadCmdSegment64):
			if cstring(data[off+8:off+24]) == "__LINKEDIT" {
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
		return nil, fmt.Errorf("%w: __LINKEDIT segment not found", ErrNotSupported)
	}
	if linkeditFileoff+linkeditFilesize != uint64(len(data)) {
		return nil, fmt.Errorf("%w: __LINKEDIT does not end at EOF (was the binary modified?)", ErrNotSupported)
	}

	// Headroom check: lcSize free bytes between the end of the LC
	// region and the first section's file payload.
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
		if seg.Filesz == 0 || seg.Offset == 0 {
			continue
		}
		if seg.Offset < firstSectOff {
			firstSectOff = seg.Offset
		}
	}
	if firstSectOff < uint64(lcEnd) || int(firstSectOff)-lcEnd < lcSize {
		return nil, fmt.Errorf("%w: not enough room for a new load command (need %d bytes, have %d)",
			ErrNotSupported, lcSize, int(firstSectOff)-lcEnd)
	}

	blobLen := uint64(len(blob))
	delta := alignUp64(blobLen, pageSize)

	// If chained fixups are present, build a patched copy that
	// accounts for our newly-inserted segment, and append it to the
	// END of __LINKEDIT (chained fixups data must live inside
	// __LINKEDIT for dyld to accept it). The original bytes are left
	// in place but become unreferenced.
	const fixupsAlign = 8
	var (
		newFixups       []byte
		newFixupsPadded uint64
	)
	if chainedLCOff >= 0 {
		patched, err := patchChainedFixupsForNewSegment(
			data[chainedDataoff:chainedDataoff+chainedSize], seaSegIdx)
		if err != nil {
			return nil, fmt.Errorf("patch chained fixups: %w", err)
		}
		newFixups = patched
		newFixupsPadded = alignUp64(uint64(len(patched)), fixupsAlign)
	}

	// linkeditExtra is deliberately equal to newFixupsPadded — only
	// 8-byte aligned, NOT page-aligned — because codesign's strict
	// validator rejects binaries with bytes after the last segment.
	linkeditExtra := newFixupsPadded

	newSegVmaddr := linkeditVmaddr
	newSegFileoff := linkeditFileoff

	// Bounds checks: the linkedit shift may push offsets past 4 GiB,
	// which silently truncates in 32-bit Mach-O fields below.
	if linkeditFileoff+delta > math.MaxUint32 {
		return nil, fmt.Errorf("%w: blob too large; would push __LINKEDIT past 4 GiB", ErrNotSupported)
	}
	if newSegFileoff > math.MaxUint32 {
		return nil, fmt.Errorf("%w: input host already past 4 GiB", ErrNotSupported)
	}

	// Build the new SEGMENT_64 + section_64 LC. segment.filesize is
	// delta (page-aligned size), not blobLen, so the file region
	// between the blob and the next segment is fully covered by
	// NODE_SEA — codesign rejects unmapped file regions.
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
	binary.LittleEndian.PutUint32(sect[48:], uint32(newSegFileoff)) // offset (bounds-checked above)

	// Build the rebuilt LC region in place within the existing slack
	// between lcEnd and firstSectOff. We keep all LCs in their
	// original order, but insert the NODE_SEA LC immediately before
	// the __LINKEDIT LC.
	totalGrowth := delta + linkeditExtra
	out := make([]byte, len(data)+int(totalGrowth))
	copy(out, data[:linkeditFileoff])

	newLCRegion := make([]byte, 0, int(sizeofcmds)+lcSize)
	newLCRegion = append(newLCRegion, data[hdrSize:linkeditLCOff]...)
	newLCRegion = append(newLCRegion, lcBuf...)
	newLCRegion = append(newLCRegion, data[linkeditLCOff:lcEnd]...)
	if hdrSize+len(newLCRegion) > int(firstSectOff) {
		return nil, fmt.Errorf("%w: rebuilt LC region exceeds first section offset", ErrNotSupported)
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

	binary.LittleEndian.PutUint32(out[ncmdsAt:], ncmds+1)
	binary.LittleEndian.PutUint32(out[sizeofcmdsAt:], sizeofcmds+uint32(lcSize))

	// Walk the new LC region and shift every linkedit_data-style file
	// offset forward by delta.
	if err := shiftLinkeditFileOffsets(out, ncmds+1, linkeditFileoff, delta); err != nil {
		return nil, err
	}

	// Write the SEA blob into the new NODE_SEA file region.
	copy(out[linkeditFileoff:], blob)
	// Page-aligned padding tail is already zero-init.

	// Write the shifted __LINKEDIT data.
	copy(out[linkeditFileoff+delta:], data[linkeditFileoff:])

	// Append patched chained-fixups blob at the end of __LINKEDIT and
	// redirect LC_DYLD_CHAINED_FIXUPS to it.
	if newFixups != nil {
		newFixupsFileoff := linkeditFileoff + delta + linkeditFilesize
		if newFixupsFileoff > math.MaxUint32 {
			return nil, fmt.Errorf("%w: chained fixups offset would exceed 4 GiB", ErrNotSupported)
		}
		copy(out[newFixupsFileoff:], newFixups)
		newChainedLCOff := chainedLCOff
		if chainedLCOff >= linkeditLCOff {
			newChainedLCOff += lcSize
		}
		binary.LittleEndian.PutUint32(out[newChainedLCOff+8:], uint32(newFixupsFileoff))
		binary.LittleEndian.PutUint32(out[newChainedLCOff+12:], uint32(len(newFixups)))
	}

	return out, nil
}

// shiftLinkeditFileOffsets walks the LC region of buf and adjusts every
// LC field that holds an absolute file offset into __LINKEDIT, adding
// delta if the original offset is at or past oldLEStart. Returns
// ErrNotSupported if any shifted offset would overflow uint32.
func shiftLinkeditFileOffsets(buf []byte, ncmds uint32, oldLEStart, delta uint64) error {
	const hdrSize = 32
	off := hdrSize
	shift := func(at int) error {
		v := uint64(binary.LittleEndian.Uint32(buf[at:]))
		if v == 0 || v < oldLEStart {
			return nil
		}
		shifted := v + delta
		if shifted > math.MaxUint32 {
			return fmt.Errorf("%w: linkedit offset would exceed 4 GiB after shift", ErrNotSupported)
		}
		binary.LittleEndian.PutUint32(buf[at:], uint32(shifted))
		return nil
	}
	for range ncmds {
		if off+8 > len(buf) {
			return fmt.Errorf("%w: malformed LC region", ErrNotSupported)
		}
		cmd := binary.LittleEndian.Uint32(buf[off:])
		sz := int(binary.LittleEndian.Uint32(buf[off+4:]))
		// All the LC types below carry absolute file offsets that
		// point into __LINKEDIT.
		var ats []int
		switch cmd {
		case 0x2: // LC_SYMTAB: symoff, stroff
			ats = []int{8, 16}
		case 0xb: // LC_DYSYMTAB: tocoff, modtaboff, extrefsymoff, indirectsymoff, extreloff, locreloff
			ats = []int{32, 40, 48, 56, 64, 72}
		case 0x22, 0x80000022: // LC_DYLD_INFO[_ONLY]: rebase/bind/weak/lazy/export off
			ats = []int{8, 16, 24, 32, 40}
		case 0x1d, // LC_CODE_SIGNATURE
			0x1e,       // LC_SEGMENT_SPLIT_INFO
			0x26,       // LC_FUNCTION_STARTS
			0x29,       // LC_DATA_IN_CODE
			0x2b,       // LC_DYLIB_CODE_SIGN_DRS
			0x2e,       // LC_LINKER_OPTIMIZATION_HINT
			0x80000033, // LC_DYLD_EXPORTS_TRIE
			0x80000034, // LC_DYLD_CHAINED_FIXUPS
			0x80000035: // LC_DYLD_CHAINED_FIXUPS variant
			ats = []int{8} // linkedit_data_command.dataoff
		}
		for _, rel := range ats {
			if err := shift(off + rel); err != nil {
				return err
			}
		}
		off += sz
	}
	return nil
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
	copy(out, orig[:arrayOff])
	binary.LittleEndian.PutUint32(out[startsOff:], segCount+1)
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
	copy(out[poolOff+4:], orig[poolOff:])
	if impOff > startsOff {
		binary.LittleEndian.PutUint32(out[8:], impOff+4)
	}
	if symOff > startsOff {
		binary.LittleEndian.PutUint32(out[12:], symOff+4)
	}
	return out, nil
}

// ---------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------

func cstring(b []byte) string {
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
