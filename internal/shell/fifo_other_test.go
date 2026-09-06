//go:build !unix

package shell_test

import "errors"

func mkfifo(string, uint32) error {
	return errors.New("mkfifo is unsupported on this platform")
}
