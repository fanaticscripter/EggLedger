//go:build !(darwin || windows)

package main

import (
	"path/filepath"

	"github.com/skratchdot/open-golang/open"
)

// hide is a noop on Linux. I don't think there's a unified way to hide files
// or directories on Linux (what does that even mean?) other than using a dot.
func hide(path string) error {
	return nil
}

// openFolderAndSelect opens the folder, since file selection depends on the
// file explorer and can't be implemented in the general case.
func openFolderAndSelect(path string) error {
	return open.Start(filepath.Dir(path))
}
