package nodesea

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// PEResourceName is the PE resource (RT_RCDATA) name Node's SEA loader
// uses with FindResource on Windows.
const PEResourceName = "NODE_SEA_BLOB"

// rtRCData is the PE resource type ID for raw data.
const rtRCData = 10

// InjectPE injects blob into the existing `.rsrc` section of the PE
// binary at path as a `NODE_SEA_BLOB` (RT_RCDATA) resource, then flips
// the SEA fuse sentinel.
//
// v1 limitations: the host must already contain a `.rsrc` section
// (always true for stock node.exe); `.rsrc` must be the last raw
// section in the file (so we can grow it without shifting other section
// data). Returns ErrNotSupported otherwise. Returns ErrAlreadyInjected
// if a NODE_SEA_BLOB resource is already present.
func InjectPE(path string, blob []byte) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	f, err := pe.NewFile(newReadSeeker(data))
	if err != nil {
		return fmt.Errorf("nodesea: parse PE: %w", err)
	}
	defer f.Close()

	if len(data) < 0x40 {
		return fmt.Errorf("%w: PE too small", ErrNotSupported)
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
		return fmt.Errorf("%w: unknown OptionalHeader magic %#x", ErrNotSupported, magic)
	}

	// Find .rsrc section.
	var rsrc *pe.Section
	rsrcIdx := -1
	for i, s := range f.Sections {
		if s.Name == ".rsrc" {
			rsrc = s
			rsrcIdx = i
		}
	}
	if rsrc == nil {
		return fmt.Errorf("%w: PE binary has no .rsrc section", ErrNotSupported)
	}

	// Check it's the last raw section.
	maxRaw := uint32(0)
	for _, s := range f.Sections {
		if s.Offset+s.Size > maxRaw {
			maxRaw = s.Offset + s.Size
		}
	}
	if rsrc.Offset+rsrc.Size != maxRaw {
		return fmt.Errorf("%w: .rsrc must be the last raw section (NODE_SEA_BLOB injection v1)", ErrNotSupported)
	}

	// Parse existing .rsrc tree.
	rsrcRaw := data[rsrc.Offset : rsrc.Offset+rsrc.Size]
	tree, err := parseResourceDir(rsrcRaw, 0, rsrc.VirtualAddress)
	if err != nil {
		return fmt.Errorf("nodesea: parse .rsrc: %w", err)
	}

	// Idempotency.
	if tree.find(rtRCData, PEResourceName) != nil {
		return ErrAlreadyInjected
	}

	// Add our entry.
	tree.add(rtRCData, PEResourceName, 0, blob)

	// Serialize. The new .rsrc will be placed at the same RVA but we
	// may need a different raw size.
	newRsrc := tree.serialize(rsrc.VirtualAddress)

	// Truncate file at end of penultimate raw section (i.e., drop the
	// old .rsrc), then write new .rsrc.
	preRsrcEnd := rsrc.Offset
	out := make([]byte, 0, int(preRsrcEnd)+len(newRsrc))
	out = append(out, data[:preRsrcEnd]...)
	out = append(out, newRsrc...)

	// Pad to FileAlignment.
	fileAlign := readFileAlignment(out, peOff)
	for uint32(len(out))%fileAlign != 0 {
		out = append(out, 0)
	}

	// Update .rsrc section header: SizeOfRawData and VirtualSize.
	shTableOff := peOff + 24 + int(binary.LittleEndian.Uint16(data[peOff+20:peOff+22]))
	shOff := shTableOff + rsrcIdx*40
	newSize := uint32(len(newRsrc))
	binary.LittleEndian.PutUint32(out[shOff+8:], newSize)                       // VirtualSize
	binary.LittleEndian.PutUint32(out[shOff+16:], uint32(len(out))-rsrc.Offset) // SizeOfRawData
	// PointerToRawData stays the same (rsrc.Offset).

	// Update DataDirectory[2] (RESOURCE) size.
	resDirEntryOff := dirOff + 2*8
	binary.LittleEndian.PutUint32(out[resDirEntryOff+4:], newSize)

	// Update SizeOfImage (must include the grown .rsrc virtually).
	// Round VirtualSize up to SectionAlignment, add to VirtualAddress.
	sectAlign := binary.LittleEndian.Uint32(out[optStart+32:])
	virtEnd := rsrc.VirtualAddress + ((newSize + sectAlign - 1) &^ (sectAlign - 1))
	// SizeOfImage at optStart+56.
	if curSize := binary.LittleEndian.Uint32(out[optStart+56:]); virtEnd > curSize {
		binary.LittleEndian.PutUint32(out[optStart+56:], virtEnd)
	}

	// Recompute checksum (zero field first, then store).
	for i := checksumOff; i < checksumOff+4; i++ {
		out[i] = 0
	}
	binary.LittleEndian.PutUint32(out[checksumOff:], peChecksum(out, checksumOff))

	tmp := path + ".inject.tmp"
	if err := os.WriteFile(tmp, out, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return FlipSentinel(path)
}

func readFileAlignment(data []byte, peOff int) uint32 {
	optStart := peOff + 24
	a := binary.LittleEndian.Uint32(data[optStart+36:])
	if a == 0 {
		return 512
	}
	return a
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
// va is the .rsrc section RVA (used to compute data RVAs).
func (d *rsrcDir) serialize(va uint32) []byte {
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

	// Allocate dir struct offsets.
	off := uint32(0)
	for _, di := range dirs {
		di.offset = off
		off += 16 + uint32(len(di.dir.entries))*8
	}
	// Allocate data-entry offsets.
	var collectLeaves func(dir *rsrcDir)
	collectLeaves = func(dir *rsrcDir) {
		for _, e := range dir.entries {
			if e.dir != nil {
				collectLeaves(e.dir)
				continue
			}
			leaves = append(leaves, &leafInfo{entry: e, entryOff: off})
			off += 16
		}
	}
	collectLeaves(d)
	// Allocate name string offsets.
	var collectNames func(dir *rsrcDir)
	collectNames = func(dir *rsrcDir) {
		for _, e := range dir.entries {
			if e.name != "" {
				names = append(names, &nameInfo{entry: e, off: off})
				off += 2 + uint32(len([]rune(e.name)))*2
			}
			if e.dir != nil {
				collectNames(e.dir)
			}
		}
	}
	collectNames(d)
	// Pad name section to 4-byte boundary before data.
	off = (off + 3) &^ 3
	// Allocate data blob offsets.
	for _, lf := range leaves {
		lf.dataOff = off
		off += uint32(len(lf.entry.data))
		off = (off + 3) &^ 3
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

	return out
}
