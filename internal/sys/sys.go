// Package sys provides small cross-platform helpers for clipboard access and
// opening URLs in a browser.
package sys

import (
	"fmt"
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

// editorCandidates are tried, in order, when neither $VISUAL nor $EDITOR is
// set. GUI editors come first because a user without $EDITOR configured is
// unlikely to want a modal terminal editor opened on top of their shell.
var editorCandidates = []string{"cursor", "code", "zed", "nvim", "vim", "nano"}

// OpenEditor opens dir in the user's editor, preferring $VISUAL then $EDITOR.
// The editor inherits the terminal, so terminal editors work and the call
// returns once the user exits them.
func OpenEditor(dir string) error {
	name, args := resolveEditor()
	if name == "" {
		return fmt.Errorf("no editor found — set $EDITOR")
	}
	cmd := exec.Command(name, append(args, dir)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

// resolveEditor splits the configured editor into its command and any flags
// ($EDITOR may be something like "code -w"), falling back to the first
// candidate found on PATH.
func resolveEditor() (string, []string) {
	for _, env := range []string{"VISUAL", "EDITOR"} {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			fields := strings.Fields(v)
			return fields[0], fields[1:]
		}
	}
	for _, c := range editorCandidates {
		if _, err := exec.LookPath(c); err == nil {
			return c, nil
		}
	}
	return "", nil
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
