package artifact

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
)

func FuzzChecksum(f *testing.F) {
	// Add some seed corpus
	f.Add("sha256", []byte("hello world"))
	f.Add("md5", []byte("test data"))
	f.Add("sha1", []byte("fuzz testing"))
	f.Add("crc32", []byte("random bytes"))
	f.Add("sha512", []byte("more data"))
	f.Add("blake2b", []byte("blake2b test"))
	f.Add("blake2s", []byte("blake2s test"))
	f.Add("sha224", []byte("sha224 data"))
	f.Add("sha384", []byte("sha384 content"))
	f.Add("sha3-256", []byte("sha3 example"))
	f.Add("sha3-512", []byte("sha3 large"))
	f.Add("sha3-224", []byte("sha3 small"))
	f.Add("sha3-384", []byte("sha3 medium"))

	f.Fuzz(func(t *testing.T, algorithm string, data []byte) {
		// Skip invalid algorithms
		validAlgorithms := map[string]bool{
			"sha256":     true,
			"md5":        true,
			"sha1":       true,
			"crc32":      true,
			"sha512":     true,
			"blake2b":    true,
			"blake2s":    true,
			"sha224":     true,
			"sha384":     true,
			"sha3-224":   true,
			"sha3-256":   true,
			"sha3-384":   true,
			"sha3-512":   true,
		}
		
		// Only test with valid algorithms to avoid expected errors
		if !validAlgorithms[algorithm] {
			t.Skip()
		}

		// Create a temporary file with fuzz data
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "fuzzfile")
		
		if err := os.WriteFile(filePath, data, 0o644); err != nil {
			t.Skip()
		}
		
		artifact := Artifact{
			Path: filePath,
		}
		
		// Calculate checksum
		_, err := artifact.Checksum(algorithm)
		if err != nil {
			// All valid algorithms should work with valid data
			t.Errorf("Checksum failed for algorithm %s: %v", algorithm, err)
		}
	})
}

func FuzzChecksumLargeData(f *testing.F) {
	f.Add("sha256", 10000)
	f.Add("md5", 50000)
	f.Add("sha1", 100000)
	
	f.Fuzz(func(t *testing.T, algorithm string, size int) {
		// Limit size to prevent excessive memory usage
		if size <= 0 || size > 1000000 {
			t.Skip()
		}
		
		// Skip invalid algorithms
		validAlgorithms := map[string]bool{
			"sha256":     true,
			"md5":        true,
			"sha1":       true,
			"crc32":      true,
			"sha512":     true,
			"blake2b":    true,
			"blake2s":    true,
			"sha224":     true,
			"sha384":     true,
			"sha3-224":   true,
			"sha3-256":   true,
			"sha3-384":   true,
			"sha3-512":   true,
		}
		
		// Only test with valid algorithms to avoid expected errors
		if !validAlgorithms[algorithm] {
			t.Skip()
		}
		
		// Generate random data of specified size
		data := make([]byte, size)
		if _, err := rand.Read(data); err != nil {
			t.Skip()
		}
		
		// Create a temporary file with fuzz data
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "largefuzzfile")
		
		if err := os.WriteFile(filePath, data, 0o644); err != nil {
			t.Skip()
		}
		
		artifact := Artifact{
			Path: filePath,
		}
		
		// Calculate checksum
		_, err := artifact.Checksum(algorithm)
		if err != nil {
			t.Errorf("Checksum failed for algorithm %s with %d bytes: %v", algorithm, size, err)
		}
	})
}