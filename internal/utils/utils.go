package utils

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// GhDashInstalled reports whether gh-dash is available in PATH.
func GhDashInstalled() bool {
	_, err := exec.LookPath("gh-dash")
	return err == nil
}

// OpenInBrowser opens url in the system default browser.
func OpenInBrowser(url string) error {
	if url == "" {
		return fmt.Errorf("no URL to open")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

// OpenInGhDash launches gh-dash. Returns (false, nil) with browser fallback if
// gh-dash is not installed — callers should surface the install hint to the user.
// Returns (true, nil) if gh-dash launched successfully.
func OpenInGhDash(url string) (launched bool, err error) {
	if !GhDashInstalled() {
		return false, OpenInBrowser(url)
	}
	cmd := exec.Command("gh-dash")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return true, cmd.Run()
}

// GhDashInstallHint returns the recommended install command for gh-dash.
func GhDashInstallHint() string {
	return "gh extension install dlvhdr/gh-dash"
}

// IsAuthError returns true if the error looks like a GitHub auth failure,
// so callers can show a friendly "run gh auth login" message.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "401") ||
		strings.Contains(msg, "authentication") ||
		strings.Contains(msg, "credentials") ||
		strings.Contains(msg, "token") ||
		strings.Contains(msg, "not logged") ||
		strings.Contains(msg, "gh auth")
}
