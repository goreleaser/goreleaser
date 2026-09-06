//go:build unix

package exec

import "syscall"

func mkfifo(name string, mode uint32) error {
	return syscall.Mkfifo(name, mode)
}
