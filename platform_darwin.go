package main

import "golang.org/x/sys/unix"

// For some reason UF_HIDDEN isn't defined in the syscall package. The value is
// thus copied from $(xcrun --show-sdk-path)/usr/include/sys/stat.h.
const UF_HIDDEN = 0x00008000

// hide hides a file or directory using chflags(2).
func hide(path string) error {
	return unix.Chflags(path, UF_HIDDEN)
}
