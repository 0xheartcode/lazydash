package utils

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

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

// OpenInGhDash launches gh-dash. Falls back to opening url in browser if not found.
func OpenInGhDash(url string) error {
	_, err := exec.LookPath("gh-dash")
	if err != nil {
		return OpenInBrowser(url)
	}
	cmd := exec.Command("gh-dash")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
