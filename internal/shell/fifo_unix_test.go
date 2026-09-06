//go:build unix

package shell_test

import "syscall"

func mkfifo(name string, mode uint32) error {
	return syscall.Mkfifo(name, mode)
}
