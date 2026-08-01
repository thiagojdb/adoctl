//go:build linux

package clipboard

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"syscall"

	"adoctl/pkg/clipboard/internal/wayland"

	atotto "github.com/atotto/clipboard"
)

// WriteMultiFormat copies content to the clipboard as both HTML (for rich-text
// apps such as Teams/Slack) and plain text (for text editors). On
// Linux/Wayland and X11 it spawns a background clipboard-owner process. If a
// clipboard-owner process cannot be started, it falls back to plain text.
func WriteMultiFormat(html, plain string) error {
	if os.Getenv("WAYLAND_DISPLAY") == "" && os.Getenv("DISPLAY") == "" {
		return atotto.WriteAll(plain)
	}

	if err := spawnClipboardServer(html, plain); err != nil {
		// Preserve a usable plain-text clipboard when the owner process cannot
		// be started. Runtime serving failures are handled by the child process.
		return atotto.WriteAll(plain)
	}
	return nil
}

// spawnClipboardServer starts the detached process that owns the clipboard.
func spawnClipboardServer(html, plain string) error {
	payload, err := json.Marshal(struct{ HTML, Plain string }{html, plain})
	if err != nil {
		return err
	}

	// Re-exec this binary as a daemonised subprocess.
	cmd := exec.Command(os.Args[0], "__clipboard-serve")
	cmd.Stdin = bytes.NewReader(payload)
	// Detach from the parent's process group so the child survives parent exit.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Start() // don't Wait — parent returns immediately
}

// ServeClipboard is called by the __clipboard-serve hidden command.
// It reads the HTML+plain payload from stdin and runs the Wayland clipboard
// owner, blocking until ownership is cancelled.
func ServeClipboard(html, plain string) error {
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		if err := serveX11Clipboard(html, plain); err != nil {
			// Keep a usable plain-text clipboard when X11 serving fails.
			return atotto.WriteAll(plain)
		}
		return nil
	}

	formats := map[string][]byte{
		"text/html":                []byte(html),
		"text/plain;charset=utf-8": []byte(plain),
		"text/plain":               []byte(plain),
		"UTF8_STRING":              []byte(plain),
		"STRING":                   []byte(plain),
	}
	if err := wayland.Serve(formats); err != nil {
		// Keep a usable plain-text clipboard when Wayland serving fails.
		return atotto.WriteAll(plain)
	}
	return nil
}
