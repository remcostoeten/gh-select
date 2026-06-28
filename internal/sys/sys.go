// Package sys provides small cross-platform helpers for clipboard access and
// opening URLs in a browser.
package sys

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Copy writes text to the system clipboard, trying the tools available on the
// host (macOS, WSL, X11, Wayland). It returns false if none are available.
func Copy(text string) bool {
	candidates := [][]string{
		{"pbcopy"},                           // macOS
		{"clip.exe"},                         // WSL
		{"wl-copy"},                          // Wayland
		{"xclip", "-selection", "clipboard"}, // X11
		{"xsel", "--clipboard", "--input"},   // X11 alt
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return true
		}
	}
	return false
}

// OpenURL opens url in the user's default browser.
func OpenURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		opener := "xdg-open"
		if _, err := exec.LookPath("wslview"); err == nil {
			opener = "wslview"
		}
		cmd = exec.Command(opener, url)
	}
	cmd.Stderr = os.Stderr
	return cmd.Start()
}
