package main

import (
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// hide hides a file or directory using SetFileAttributes.
func hide(path string) error {
	u16ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return windows.SetFileAttributes(u16ptr, windows.FILE_ATTRIBUTE_HIDDEN)
}

func openFolderAndSelect(path string) error {
	abspath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	cmd := exec.Command("explorer.exe", "/select,", abspath)
	return cmd.Start()
}
