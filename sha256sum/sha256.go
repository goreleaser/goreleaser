package sha256sum

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// For calculates the SHA256 sum for the given file
func For(path string) (result string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return
	}

	result = hex.EncodeToString(hash.Sum(nil))
	return
}
