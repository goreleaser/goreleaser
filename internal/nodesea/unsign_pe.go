package nodesea

import (
	"debug/pe"
	"encoding/binary"
	"fmt"
	"os"
)

// UnsignPE removes the Authenticode certificate table from a PE binary,
// zeroes the corresponding data directory entry, and recomputes the
// OptionalHeader checksum.
//
// Like UnsignMachO this is conservative: the certificate table must be
// the last thing in the file. If anything follows it the function
// rejects the binary with ErrNotSupported. If there is no certificate
// table the function is a no-op.
func UnsignPE(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	f, err := pe.NewFile(newReadSeeker(data))
	if err != nil {
		return fmt.Errorf("nodesea: parse PE: %w", err)
	}
	defer f.Close()

	// Find e_lfanew (offset of the PE signature).
	if len(data) < 0x40 {
		return fmt.Errorf("%w: PE too small", ErrNotSupported)
	}
	peOff := int(binary.LittleEndian.Uint32(data[0x3c:0x40]))
	// "PE\0\0" + IMAGE_FILE_HEADER (20 bytes) → optional header begins
	// at peOff + 4 + 20.
	optStart := peOff + 24
	if optStart >= len(data) {
		return fmt.Errorf("%w: PE truncated", ErrNotSupported)
	}

	magic := binary.LittleEndian.Uint16(data[optStart : optStart+2])
	var (
		checksumOff int
		dirOff      int // offset of DataDirectory[0]
	)
	switch magic {
	case 0x10b: // PE32
		checksumOff = optStart + 64
		dirOff = optStart + 96
	case 0x20b: // PE32+
		checksumOff = optStart + 64
		dirOff = optStart + 112
	default:
		return fmt.Errorf("%w: unknown OptionalHeader magic %#x", ErrNotSupported, magic)
	}

	// DataDirectory[4] is IMAGE_DIRECTORY_ENTRY_SECURITY.
	const secIdx = 4
	secEntryOff := dirOff + secIdx*8
	if secEntryOff+8 > len(data) {
		return fmt.Errorf("%w: PE truncated before DataDirectory[4]", ErrNotSupported)
	}

	va := binary.LittleEndian.Uint32(data[secEntryOff : secEntryOff+4])
	size := binary.LittleEndian.Uint32(data[secEntryOff+4 : secEntryOff+8])

	if va == 0 && size == 0 {
		// No signature; nothing to do.
		return nil
	}

	// va is a *file offset* (not RVA) for the security directory.
	if uint64(va)+uint64(size) != uint64(len(data)) {
		return fmt.Errorf("%w: certificate table is not at end of file", ErrNotSupported)
	}

	// Zero the directory entry.
	for i := secEntryOff; i < secEntryOff+8; i++ {
		data[i] = 0
	}

	// Truncate.
	data = data[:va]

	// Recompute checksum.
	binary.LittleEndian.PutUint32(data[checksumOff:checksumOff+4], 0)
	cks := peChecksum(data, checksumOff)
	binary.LittleEndian.PutUint32(data[checksumOff:checksumOff+4], cks)

	tmp := path + ".unsign.tmp"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// peChecksum implements the PE OptionalHeader checksum algorithm:
//
//	sum = sum of all 16-bit little-endian words in the file with the
//	      checksum field treated as 0
//	sum = (sum & 0xffff) + (sum >> 16)   repeatedly until it fits in 16 bits
//	checksum = sum + file size
//
// `data` should already have the checksum field zeroed.
func peChecksum(data []byte, checksumOff int) uint32 {
	var sum uint64
	limit := len(data) &^ 1
	for i := 0; i < limit; i += 2 {
		// Skip the 4 bytes of the checksum field — caller has already
		// zeroed them, but be explicit anyway in case it wasn't.
		if i == checksumOff || i == checksumOff+2 {
			continue
		}
		w := uint64(binary.LittleEndian.Uint16(data[i : i+2]))
		sum += w
		sum = (sum & 0xffff) + (sum >> 16)
	}
	if len(data)&1 == 1 {
		sum += uint64(data[len(data)-1])
		sum = (sum & 0xffff) + (sum >> 16)
	}
	sum = (sum & 0xffff) + (sum >> 16)
	return uint32(sum) + uint32(len(data))
}
