//go:build !darwin && !windows

package cmd

import (
	"os"
	"os/exec"
	"strings"
)

func guiAvailable() bool {
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

func nativePickFolder(title, startDir string) (string, bool) {
	start := startDir
	if start == "" {
		start = "/"
	}
	for _, args := range [][]string{
		{"zenity", "--file-selection", "--directory", "--filename", start, "--title", title},
		{"kdialog", "--getexistingdirectory", start, "--title", title},
	} {
		out, err := exec.Command(args[0], args[1:]...).Output()
		if err == nil {
			if s := strings.TrimSpace(string(out)); s != "" {
				return s, true
			}
		}
	}
	return "", false
}
