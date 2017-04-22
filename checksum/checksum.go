// Package checksum contain algorithms to checksum files
package checksum

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

// SHA256 sum of the given file
func SHA256(path string) (result string, err error) {
	return calculate(sha256.New(), path)
}

func calculate(hash hash.Hash, path string) (result string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	return doCalculate(hash, file)
}

func doCalculate(hash hash.Hash, file *os.File) (result string, err error) {
	_, err = io.Copy(hash, file)
	if err != nil {
		return
	}

	result = hex.EncodeToString(hash.Sum(nil))
	return
}
