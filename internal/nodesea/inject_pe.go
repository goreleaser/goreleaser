package nodesea

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

// PEResourceName is the PE resource (RT_RCDATA) name Node's SEA loader
// uses with FindResource on Windows.
const PEResourceName = "NODE_SEA_BLOB"

// rtRCData is the PE resource type ID for raw data.
const rtRCData = 10

// injectPEBytes returns data with blob spliced in as a `NODE_SEA_BLOB`
// (RT_RCDATA) resource, and the SEA fuse sentinel flipped from `:0`
// to `:1`.
//
// The input is expected to be already unsigned (no Authenticode
// certificate table). Pass it through unsignPEBytes first if it
// might be signed — appending a section past the cert table would
// invalidate any existing signature anyway.
//
// Strategy: parse the existing `.rsrc` tree (so we preserve
// version-info, icon, etc.), add our `NODE_SEA_BLOB` entry, then write
// the fully serialized tree as a brand-new section appended at
// end-of-file. The resource data directory is repointed at the new
// section. The original `.rsrc` bytes stay mapped but become
// unreferenced (the loader uses the data-directory pointer for
// resource lookup).
//
// This approach handles real-world `node.exe`, where `.rsrc` is not
// the last raw section (`.reloc` follows it). Returns ErrNotSupported
// when the binary lacks a `.rsrc` section, or when there is not
// enough slack in the section header table for one more entry.
// Returns ErrAlreadyInjected when a `NODE_SEA_BLOB` resource is
// already present.
func injectPEBytes(data, blob []byte) ([]byte, error) {
	f, err := pe.NewFile(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("nodesea: parse PE: %w", err)
	}
	defer f.Close()

	if len(data) < 0x40 {
		return nil, fmt.Errorf("%w: PE too small", ErrNotSupported)
	}
	peOff := int(binary.LittleEndian.Uint32(data[0x3c:0x40]))
	optStart := peOff + 24

	magic := binary.LittleEndian.Uint16(data[optStart : optStart+2])
	var (
		checksumOff int
		dirOff      int
	)
	switch magic {
	case 0x10b:
		checksumOff = optStart + 64
		dirOff = optStart + 96
	case 0x20b:
		checksumOff = optStart + 64
		dirOff = optStart + 112
	default:
		return nil, fmt.Errorf("%w: unknown OptionalHeader magic %#x", ErrNotSupported, magic)
	}

	sectAlign := binary.LittleEndian.Uint32(data[optStart+32:])
	fileAlign := binary.LittleEndian.Uint32(data[optStart+36:])
	if sectAlign == 0 {
		sectAlign = 0x1000
	}
	if fileAlign == 0 {
		fileAlign = 512
	}

	// Find the section currently pointed at by DataDirectory[2] (which
	// may not be ".rsrc" — e.g. on a binary we already injected, the
	// data directory points at our appended ".sea" section).
	resDirEntryOff := dirOff + 2*8
	resVA := binary.LittleEndian.Uint32(data[resDirEntryOff:])
	resSize := binary.LittleEndian.Uint32(data[resDirEntryOff+4:])
	var rsrc *pe.Section
	if resVA != 0 {
		for _, s := range f.Sections {
			if resVA >= s.VirtualAddress && resVA < s.VirtualAddress+s.VirtualSize {
				rsrc = s
				break
			}
		}
	}
	if rsrc == nil {
		// Fall back to the conventional name.
		for _, s := range f.Sections {
			if s.Name == ".rsrc" {
				rsrc = s
				resVA = s.VirtualAddress
				resSize = s.VirtualSize
				break
			}
		}
	}
	if rsrc == nil {
		return nil, fmt.Errorf("%w: PE binary has no .rsrc section", ErrNotSupported)
	}

	// Parse existing resource tree so we preserve existing resources
	// (version info, icon, manifest, …).
	resOffInSect := resVA - rsrc.VirtualAddress
	if resOffInSect+resSize > rsrc.Size {
		return nil, fmt.Errorf("%w: resource directory exceeds section bounds", ErrNotSupported)
	}
	rsrcRaw := data[rsrc.Offset+resOffInSect : rsrc.Offset+resOffInSect+resSize]
	tree, err := parseResourceDir(rsrcRaw, 0, resVA)
	if err != nil {
		return nil, fmt.Errorf("nodesea: parse .rsrc: %w", err)
	}

	// Idempotency.
	if tree.find(rtRCData, PEResourceName) != nil {
		return nil, ErrAlreadyInjected
	}

	tree.add(rtRCData, PEResourceName, 0, blob)

	// Compute the VA for our new section: just past the highest virtual
	// extent of any existing section, rounded up to SectionAlignment.
	maxVirtEnd := uint32(0)
	for _, s := range f.Sections {
		end := s.VirtualAddress + s.VirtualSize
		if end > maxVirtEnd {
			maxVirtEnd = end
		}
	}
	newVA := alignUp(maxVirtEnd, sectAlign)

	// Serialize new resource tree at the new VA.
	newRsrc, err := tree.serialize(newVA)
	if err != nil {
		return nil, err
	}
	newRsrcSize := uint32(len(newRsrc))

	// Pad raw size to FileAlignment.
	paddedSize := alignUp(newRsrcSize, fileAlign)

	// Verify there's room in the section header table for one more entry.
	sizeOfOpt := int(binary.LittleEndian.Uint16(data[peOff+20 : peOff+22]))
	numSections := binary.LittleEndian.Uint16(data[peOff+6 : peOff+8])
	sectTableOff := peOff + 24 + sizeOfOpt
	newSectHeaderOff := sectTableOff + int(numSections)*40

	firstRaw := uint32(len(data))
	for _, s := range f.Sections {
		if s.Offset > 0 && s.Offset < firstRaw {
			firstRaw = s.Offset
		}
	}
	if uint32(newSectHeaderOff)+40 > firstRaw {
		return nil, fmt.Errorf("%w: no header slack for new section (need 40 bytes, have %d)",
			ErrNotSupported, int(firstRaw)-newSectHeaderOff)
	}

	// Build output: original data + padded new section payload.
	// Pad the original tail to FileAlignment first so our new raw offset
	// sits on a FileAlignment boundary (PE requirement).
	out := make([]byte, 0, len(data)+int(paddedSize)+int(fileAlign))
	out = append(out, data...)
	for uint32(len(out))%fileAlign != 0 {
		out = append(out, 0)
	}
	newRaw := uint32(len(out))
	out = append(out, newRsrc...)
	for uint32(len(out))%fileAlign != 0 {
		out = append(out, 0)
	}

	// Section header (40 bytes): Name[8], VirtualSize, VirtualAddress,
	// SizeOfRawData, PointerToRawData, PointerToRelocations,
	// PointerToLinenumbers, NumberOfRelocations, NumberOfLinenumbers,
	// Characteristics.
	const (
		imageScnCntInitData = 0x00000040
		imageScnMemRead     = 0x40000000
	)
	hdr := make([]byte, 40)
	copy(hdr[0:8], ".sea\x00\x00\x00\x00")
	binary.LittleEndian.PutUint32(hdr[8:], newRsrcSize)
	binary.LittleEndian.PutUint32(hdr[12:], newVA)
	binary.LittleEndian.PutUint32(hdr[16:], paddedSize)
	binary.LittleEndian.PutUint32(hdr[20:], newRaw)
	// PointerToRelocations, PointerToLinenumbers, counts: zero.
	binary.LittleEndian.PutUint32(hdr[36:], imageScnCntInitData|imageScnMemRead)
	copy(out[newSectHeaderOff:newSectHeaderOff+40], hdr)

	// Update NumberOfSections.
	binary.LittleEndian.PutUint16(out[peOff+6:], numSections+1)

	// Update DataDirectory[2] (resource): RVA + Size.
	binary.LittleEndian.PutUint32(out[resDirEntryOff:], newVA)
	binary.LittleEndian.PutUint32(out[resDirEntryOff+4:], newRsrcSize)

	// Update SizeOfImage = newVA + alignUp(newRsrcSize, sectAlign).
	newSizeOfImage := newVA + alignUp(newRsrcSize, sectAlign)
	binary.LittleEndian.PutUint32(out[optStart+56:], newSizeOfImage)

	// Recompute checksum (zero field first, then store).
	for i := checksumOff; i < checksumOff+4; i++ {
		out[i] = 0
	}
	binary.LittleEndian.PutUint32(out[checksumOff:], peChecksum(out, checksumOff))

	return flipSentinel(out)
}

func alignUp(v, a uint32) uint32 {
	if a == 0 {
		return v
	}
	return (v + a - 1) &^ (a - 1)
}

// --- resource tree model ---

type rsrcEntry struct {
	id   uint32 // used if name == ""
	name string // unicode name; if non-empty, used instead of id
	// Either dir or data is non-nil.
	dir  *rsrcDir
	data []byte
}

type rsrcDir struct {
	entries []*rsrcEntry
}

func (d *rsrcDir) find(typeID uint32, name string) *rsrcEntry {
	for _, e := range d.entries {
		if e.name == "" && e.id == typeID && e.dir != nil {
			for _, ne := range e.dir.entries {
				if ne.name == name {
					return ne
				}
			}
		}
	}
	return nil
}

func (d *rsrcDir) add(typeID uint32, name string, lang uint32, data []byte) {
	// Find or create the type-level dir.
	var typeDir *rsrcEntry
	for _, e := range d.entries {
		if e.name == "" && e.id == typeID && e.dir != nil {
			typeDir = e
			break
		}
	}
	if typeDir == nil {
		typeDir = &rsrcEntry{id: typeID, dir: &rsrcDir{}}
		d.entries = append(d.entries, typeDir)
		sortDirEntries(d.entries)
	}
	// Find or create the name-level dir.
	var nameDir *rsrcEntry
	for _, e := range typeDir.dir.entries {
		if e.name == name {
			nameDir = e
			break
		}
	}
	if nameDir == nil {
		nameDir = &rsrcEntry{name: name, dir: &rsrcDir{}}
		typeDir.dir.entries = append(typeDir.dir.entries, nameDir)
		sortDirEntries(typeDir.dir.entries)
	}
	// Add the language leaf.
	leaf := &rsrcEntry{id: lang, data: data}
	nameDir.dir.entries = append(nameDir.dir.entries, leaf)
	sortDirEntries(nameDir.dir.entries)
}

// sortDirEntries: per PE spec, named entries first sorted by name
// (case-insensitive ascending), then ID entries sorted by ID ascending.
func sortDirEntries(es []*rsrcEntry) {
	sort.SliceStable(es, func(i, j int) bool {
		ai, aj := es[i], es[j]
		if (ai.name == "") != (aj.name == "") {
			// named first
			return ai.name != ""
		}
		if ai.name != "" {
			return strings.ToLower(ai.name) < strings.ToLower(aj.name)
		}
		return ai.id < aj.id
	})
}

// parseResourceDir parses a PE resource directory rooted at `dirOff`
// inside `data`. `va` is the .rsrc section's virtual address (used to
// translate data entry RVAs).
func parseResourceDir(data []byte, dirOff uint32, va uint32) (*rsrcDir, error) {
	if int(dirOff)+16 > len(data) {
		return nil, errors.New("dir header out of range")
	}
	nameCount := binary.LittleEndian.Uint16(data[dirOff+12:])
	idCount := binary.LittleEndian.Uint16(data[dirOff+14:])
	total := int(nameCount) + int(idCount)

	d := &rsrcDir{}
	entryOff := int(dirOff) + 16
	for i := range total {
		base := entryOff + i*8
		if base+8 > len(data) {
			return nil, errors.New("entry out of range")
		}
		nameOrID := binary.LittleEndian.Uint32(data[base:])
		offsetField := binary.LittleEndian.Uint32(data[base+4:])

		e := &rsrcEntry{}
		if nameOrID&0x80000000 != 0 {
			// name
			nameOff := nameOrID & 0x7fffffff
			if int(nameOff)+2 > len(data) {
				return nil, errors.New("name out of range")
			}
			n := binary.LittleEndian.Uint16(data[nameOff:])
			if int(nameOff)+2+int(n)*2 > len(data) {
				return nil, errors.New("name body out of range")
			}
			runes := make([]rune, n)
			for k := 0; k < int(n); k++ {
				runes[k] = rune(binary.LittleEndian.Uint16(data[int(nameOff)+2+k*2:]))
			}
			e.name = string(runes)
		} else {
			e.id = nameOrID
		}
		if offsetField&0x80000000 != 0 {
			subDirOff := offsetField & 0x7fffffff
			sub, err := parseResourceDir(data, subDirOff, va)
			if err != nil {
				return nil, err
			}
			e.dir = sub
		} else {
			// data entry: 16 bytes (DataRVA, Size, Codepage, Reserved)
			deOff := offsetField
			if int(deOff)+16 > len(data) {
				return nil, errors.New("data entry out of range")
			}
			rva := binary.LittleEndian.Uint32(data[deOff:])
			size := binary.LittleEndian.Uint32(data[deOff+4:])
			fileOff := rva - va
			if int(fileOff)+int(size) > len(data) {
				return nil, errors.New("data body out of range")
			}
			e.data = bytes.Clone(data[fileOff : fileOff+size])
		}
		d.entries = append(d.entries, e)
	}
	return d, nil
}

// serialize emits the .rsrc section bytes. Layout:
//
//	[directories...] [data entries...] [name strings...] [data blobs...]
//
// va is the .rsrc section RVA (used to compute data RVAs). Returns
// ErrNotSupported when any computed offset would overflow uint32 or
// when a resource name would exceed the 16-bit length encoded in the
// IMAGE_RESOURCE_DIR_STRING_U header — both are silent-truncation
// hazards otherwise.
func (d *rsrcDir) serialize(va uint32) ([]byte, error) {
	// First pass: collect every directory, name string, data entry, and
	// data blob to compute layout offsets.
	type dirInfo struct {
		dir    *rsrcDir
		offset uint32
	}
	type leafInfo struct {
		entry    *rsrcEntry
		entryOff uint32 // offset of data-entry struct
		dataOff  uint32 // offset of the data bytes
	}

	var dirs []*dirInfo
	var leaves []*leafInfo
	type nameInfo struct {
		entry *rsrcEntry
		off   uint32
	}
	var names []*nameInfo

	// Walk in BFS to allocate dir struct space first (consistent with
	// linker output convention).
	queue := []*rsrcDir{d}
	dirs = append(dirs, &dirInfo{dir: d})
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, e := range cur.entries {
			if e.dir != nil {
				dirs = append(dirs, &dirInfo{dir: e.dir})
				queue = append(queue, e.dir)
			}
		}
	}

	// Compute and bounds-check every offset in uint64 first; the .rsrc
	// section RVA + offset must fit in 32 bits because that's what the
	// IMAGE_RESOURCE_DATA_ENTRY.OffsetToData field stores.
	off := uint64(0)
	bumpOff := func(by uint64) error {
		off += by
		if off > math.MaxUint32 {
			return fmt.Errorf("%w: .rsrc serialized layout exceeds 4 GiB", ErrNotSupported)
		}
		return nil
	}

	// Allocate dir struct offsets.
	for _, di := range dirs {
		di.offset = uint32(off)
		if err := bumpOff(16 + uint64(len(di.dir.entries))*8); err != nil {
			return nil, err
		}
	}
	// Allocate data-entry offsets.
	var collectLeavesErr error
	var collectLeaves func(dir *rsrcDir)
	collectLeaves = func(dir *rsrcDir) {
		if collectLeavesErr != nil {
			return
		}
		for _, e := range dir.entries {
			if e.dir != nil {
				collectLeaves(e.dir)
				continue
			}
			leaves = append(leaves, &leafInfo{entry: e, entryOff: uint32(off)})
			if err := bumpOff(16); err != nil {
				collectLeavesErr = err
				return
			}
		}
	}
	collectLeaves(d)
	if collectLeavesErr != nil {
		return nil, collectLeavesErr
	}
	// Allocate name string offsets. PE encodes the name length as a
	// uint16 right before the UTF-16 code units, so names longer than
	// 0xFFFF would silently wrap.
	var collectNamesErr error
	var collectNames func(dir *rsrcDir)
	collectNames = func(dir *rsrcDir) {
		if collectNamesErr != nil {
			return
		}
		for _, e := range dir.entries {
			if e.name != "" {
				runes := []rune(e.name)
				if len(runes) > math.MaxUint16 {
					collectNamesErr = fmt.Errorf(
						"%w: resource name %q has %d code points (PE limit is %d)",
						ErrNotSupported, e.name, len(runes), math.MaxUint16)
					return
				}
				names = append(names, &nameInfo{entry: e, off: uint32(off)})
				if err := bumpOff(2 + uint64(len(runes))*2); err != nil {
					collectNamesErr = err
					return
				}
			}
			if e.dir != nil {
				collectNames(e.dir)
			}
		}
	}
	collectNames(d)
	if collectNamesErr != nil {
		return nil, collectNamesErr
	}
	// Pad name section to 4-byte boundary before data.
	off = (off + 3) &^ 3
	// Allocate data blob offsets.
	for _, lf := range leaves {
		lf.dataOff = uint32(off)
		if err := bumpOff(uint64(len(lf.entry.data))); err != nil {
			return nil, err
		}
		off = (off + 3) &^ 3
	}
	// And finally check that va + the highest data-entry offset still
	// fits in 32 bits (DataRVA is uint32).
	if uint64(va)+off > math.MaxUint32 {
		return nil, fmt.Errorf("%w: .rsrc RVA + offset would exceed 4 GiB", ErrNotSupported)
	}
	totalSize := off

	// Now emit.
	out := make([]byte, totalSize)

	dirOf := func(dir *rsrcDir) uint32 {
		for _, di := range dirs {
			if di.dir == dir {
				return di.offset
			}
		}
		panic("dir not found")
	}
	leafOf := func(e *rsrcEntry) *leafInfo {
		for _, lf := range leaves {
			if lf.entry == e {
				return lf
			}
		}
		panic("leaf not found")
	}
	nameOf := func(e *rsrcEntry) uint32 {
		for _, ni := range names {
			if ni.entry == e {
				return ni.off
			}
		}
		panic("name not found")
	}

	for _, di := range dirs {
		base := di.offset
		// IMAGE_RESOURCE_DIRECTORY (16 bytes): chars(4), timedate(4),
		// majorVer(2), minorVer(2), nNamed(2), nID(2)
		var nNamed, nID uint16
		for _, e := range di.dir.entries {
			if e.name != "" {
				nNamed++
			} else {
				nID++
			}
		}
		binary.LittleEndian.PutUint16(out[base+12:], nNamed)
		binary.LittleEndian.PutUint16(out[base+14:], nID)
		// Entries (ensure sorted; sortDirEntries was applied on add).
		entryBase := base + 16
		for i, e := range di.dir.entries {
			eb := entryBase + uint32(i)*8
			if e.name != "" {
				binary.LittleEndian.PutUint32(out[eb:], nameOf(e)|0x80000000)
			} else {
				binary.LittleEndian.PutUint32(out[eb:], e.id)
			}
			if e.dir != nil {
				binary.LittleEndian.PutUint32(out[eb+4:], dirOf(e.dir)|0x80000000)
			} else {
				binary.LittleEndian.PutUint32(out[eb+4:], leafOf(e).entryOff)
			}
		}
	}

	// Write data entries.
	for _, lf := range leaves {
		eo := lf.entryOff
		binary.LittleEndian.PutUint32(out[eo:], va+lf.dataOff)                // DataRVA
		binary.LittleEndian.PutUint32(out[eo+4:], uint32(len(lf.entry.data))) // Size
		binary.LittleEndian.PutUint32(out[eo+8:], 0)                          // Codepage
		binary.LittleEndian.PutUint32(out[eo+12:], 0)                         // Reserved
	}

	// Write name strings.
	for _, ni := range names {
		runes := []rune(ni.entry.name)
		binary.LittleEndian.PutUint16(out[ni.off:], uint16(len(runes)))
		for k, r := range runes {
			binary.LittleEndian.PutUint16(out[ni.off+2+uint32(k)*2:], uint16(r))
		}
	}

	// Write data blobs.
	for _, lf := range leaves {
		copy(out[lf.dataOff:], lf.entry.data)
	}

	return out, nil
}
