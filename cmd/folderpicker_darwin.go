//go:build darwin

package cmd

import (
	"os/exec"
	"path/filepath"
	"strings"
)

func guiAvailable() bool { return !isSSHSession() }

func nativePickFolder(title, startDir string) (string, bool) {
	safe := strings.ReplaceAll(title, `"`, `\"`)
	script := `POSIX path of (choose folder with prompt "` + safe + `"`
	if startDir != "" {
		script += ` default location POSIX file "` + strings.ReplaceAll(startDir, `"`, `\"`) + `"`
	}
	script += `)`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", false
	}
	path := filepath.Clean(strings.TrimSpace(string(out)))
	if path == "" || path == "." {
		return "", false
	}
	return path, true
}
