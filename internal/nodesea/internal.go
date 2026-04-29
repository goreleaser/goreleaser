package nodesea

// alignUp64 rounds v up to the nearest multiple of align (which must
// be a power of two).
func alignUp64(v, align uint64) uint64 {
	return (v + align - 1) &^ (align - 1)
}
