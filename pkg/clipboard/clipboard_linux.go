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
// Linux/Wayland it spawns a background clipboard-owner process; on X11 it
// falls back to plain text only.
func WriteMultiFormat(html, plain string) error {
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		// X11 fallback: plain text only.
		return atotto.WriteAll(plain)
	}
	return spawnClipboardServer(html, plain)
}

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
	formats := map[string][]byte{
		"text/html":                []byte(html),
		"text/plain;charset=utf-8": []byte(plain),
		"text/plain":               []byte(plain),
		"UTF8_STRING":              []byte(plain),
		"STRING":                   []byte(plain),
	}
	return wayland.Serve(formats)
}
