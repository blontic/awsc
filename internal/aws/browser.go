package aws

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// OpenBrowser opens the specified URL in the default web browser
// Supports macOS, Linux, WSL, and Windows
func OpenBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", "", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // "linux", "freebsd", "openbsd", "netbsd"
		// Check if running in WSL
		if IsWSL() {
			// Use Windows browser from WSL via rundll32 which handles URLs better
			cmd = "rundll32.exe"
			args = []string{"url.dll,FileProtocolHandler", url}
		} else {
			cmd = "xdg-open"
			args = []string{url}
		}
	}
	return exec.Command(cmd, args...).Start()
}

// IsWSL checks if the current environment is Windows Subsystem for Linux
func IsWSL() bool {
	// Check for WSL-specific environment variables
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSL_INTEROP") != "" {
		return true
	}

	// Check /proc/version for Microsoft/WSL
	if data, err := os.ReadFile("/proc/version"); err == nil {
		version := strings.ToLower(string(data))
		if strings.Contains(version, "microsoft") || strings.Contains(version, "wsl") {
			return true
		}
	}

	return false
}
