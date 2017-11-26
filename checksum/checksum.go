// Package checksum contain algorithms to checksum files
package checksum

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"os"

	"github.com/apex/log"
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
	defer func() {
		if err := file.Close(); err != nil {
			log.WithError(err).Errorf("failed to close %s", path)
		}
	}()

	return doCalculate(hash, file)
}

func doCalculate(hash hash.Hash, file io.Reader) (result string, err error) {
	_, err = io.Copy(hash, file)
	if err != nil {
		return
	}

	result = hex.EncodeToString(hash.Sum(nil))
	return
}
