//go:build windows

package format

import (
	"os"
	"syscall"
)

// enableVirtualTerminalProcessing is the Windows console mode flag that makes
// the console interpret ANSI escape sequences instead of printing them
// literally. Supported on Windows 10 version 1511 (November 2015) and later.
const enableVirtualTerminalProcessing = 0x0004

// SetConsoleMode is not exported from stdlib syscall on Windows, so bind it
// via LazyDLL. Pure stdlib, no external modules.
var procSetConsoleMode = syscall.NewLazyDLL("kernel32.dll").NewProc("SetConsoleMode")

// setConsoleMode wraps the Win32 SetConsoleMode call.
// Returns a non-nil error if the call fails.
func setConsoleMode(handle syscall.Handle, mode uint32) error {
	ret, _, callErr := procSetConsoleMode.Call(uintptr(handle), uintptr(mode))
	if ret == 0 {
		return callErr
	}
	return nil
}

// IsTTY reports whether f is a console that can render ANSI escape sequences.
// On Windows this also enables ENABLE_VIRTUAL_TERMINAL_PROCESSING when needed,
// so callers can emit the same ANSI codes used on Unix.
func IsTTY(f *os.File) bool {
	info, err := f.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	handle := syscall.Handle(f.Fd())
	var mode uint32
	if err := syscall.GetConsoleMode(handle, &mode); err != nil {
		// Not a real Windows console (e.g. mintty/MSYS). Caller must not
		// emit ANSI — it would print as literal text.
		return false
	}
	if mode&enableVirtualTerminalProcessing != 0 {
		return true
	}
	if err := setConsoleMode(handle, mode|enableVirtualTerminalProcessing); err != nil {
		// Legacy conhost without VT support (pre-Win10 1511).
		return false
	}
	return true
}
