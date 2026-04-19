//go:build !windows

package format

import "os"

// IsTTY reports whether f is a terminal (character device).
func IsTTY(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
