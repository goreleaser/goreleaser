package artifact

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func FuzzChecksum(f *testing.F) {
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
		if !validAlgorithms[algorithm] {
			t.Skip()
		}

		filePath := filepath.Join(t.TempDir(), "fuzzfile")
		require.NoError(t, os.WriteFile(filePath, data, 0o644))
		artifact := Artifact{
			Path: filePath,
		}
		_, err := artifact.Checksum(algorithm)
		require.NoError(t, err)
	})
}

func FuzzChecksumLargeData(f *testing.F) {
	f.Add("sha256", 10000)
	f.Add("md5", 50000)
	f.Add("sha1", 100000)

	f.Fuzz(func(t *testing.T, algorithm string, size int) {
		if !validAlgorithms[algorithm] {
			t.Skip()
		}
		data := make([]byte, size)
		_, err := rand.Read(data)
		require.NoError(t, err)

		filePath := filepath.Join(t.TempDir(), "largefuzzfile")
		require.NoError(t, os.WriteFile(filePath, data, 0o644))
		artifact := Artifact{
			Path: filePath,
		}

		// Calculate checksum
		_, err = artifact.Checksum(algorithm)
		require.NoError(t, err)
	})
}

var validAlgorithms = map[string]bool{
	"sha256":   true,
	"md5":      true,
	"sha1":     true,
	"crc32":    true,
	"sha512":   true,
	"blake2b":  true,
	"blake2s":  true,
	"sha224":   true,
	"sha384":   true,
	"sha3-224": true,
	"sha3-256": true,
	"sha3-384": true,
	"sha3-512": true,
}
