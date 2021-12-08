//go:build !(darwin || windows)

package main

// hide is a noop on Linux. I don't think there's a unified way to hide files
// or directories on Linux (what does that even mean?) other than using a dot.
func hide(path string) error {
	return nil
}
