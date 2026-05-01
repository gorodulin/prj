package cmd

import "os"

// resolveStartDir returns startDir when it exists, otherwise the user's home
// directory. Picker dialogs error or behave poorly when handed a missing path.
func resolveStartDir(startDir string) string {
	if startDir != "" {
		if _, err := os.Stat(startDir); err == nil {
			return startDir
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return ""
}

// isSSHSession reports whether this process is running under an sshd-spawned
// shell. Used by platforms where SSH is the dominant cause of an unreachable
// GUI (macOS, Windows).
func isSSHSession() bool {
	return os.Getenv("SSH_CONNECTION") != "" ||
		os.Getenv("SSH_CLIENT") != "" ||
		os.Getenv("SSH_TTY") != ""
}

// guiPickFolder opens a native folder dialog and returns the chosen path,
// or ("", false) when no GUI is reachable. The per-platform guiAvailable
// predicate gates the call; nativePickFolder performs the dialog itself.
func guiPickFolder(title, startDir string) (string, bool) {
	if !guiAvailable() {
		return "", false
	}
	return nativePickFolder(title, resolveStartDir(startDir))
}
