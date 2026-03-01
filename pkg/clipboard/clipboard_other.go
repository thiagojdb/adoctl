//go:build !linux

package clipboard

import atotto "github.com/atotto/clipboard"

// WriteMultiFormat copies content to the clipboard. On non-Linux platforms
// only plain text is supported.
func WriteMultiFormat(html, plain string) error {
	return atotto.WriteAll(plain)
}

// ServeClipboard is not used on non-Linux platforms.
func ServeClipboard(html, plain string) error {
	return nil
}
