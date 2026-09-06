//go:build !unix

package exec

import "errors"

func mkfifo(string, uint32) error {
	return errors.New("mkfifo is unsupported on this platform")
}
