package utils

import (
	"os/exec"
	"runtime"
)

// OpenBrowser attempts to open a URL in the default browser based on the operating system.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return nil
	}
	return cmd.Start()
}
