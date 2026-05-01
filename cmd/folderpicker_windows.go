//go:build windows

package cmd

import (
	"os/exec"
	"strings"
)

func guiAvailable() bool { return !isSSHSession() }

func nativePickFolder(title, startDir string) (string, bool) {
	safe := strings.ReplaceAll(title, "'", "''")
	safeDir := strings.ReplaceAll(startDir, "'", "''")
	script := `Add-Type -AssemblyName System.Windows.Forms;` +
		`$f=New-Object System.Windows.Forms.FolderBrowserDialog;` +
		`$f.Description='` + safe + `';`
	if startDir != "" {
		script += `$f.SelectedPath='` + safeDir + `';`
	}
	script += `if($f.ShowDialog() -eq 'OK'){$f.SelectedPath}`
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return "", false
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return "", false
	}
	return path, true
}
