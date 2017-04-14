package checksum

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

// MD5 sum of the given file
func MD5(path string) (result string, err error) {
	return calculate(md5.New(), path)
}

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

	_, err = io.Copy(hash, file)
	if err != nil {
		return
	}

	result = hex.EncodeToString(hash.Sum(nil))
	return
}
