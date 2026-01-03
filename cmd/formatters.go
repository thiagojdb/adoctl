package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"adoctl/pkg/clipboard"

	atottoclipboard "github.com/atotto/clipboard"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// OutputFormat represents the output format type
type OutputFormat string

const (
	// FormatTable is the default human-readable table format
	FormatTable OutputFormat = "table"
	// FormatModern is the modern table format with icons
	FormatModern OutputFormat = "modern"
	// FormatJSON outputs as JSON
	FormatJSON OutputFormat = "json"
	// FormatYAML outputs as YAML
	FormatYAML OutputFormat = "yaml"
)

// OutputWriter handles structured output formatting
type OutputWriter struct {
	format OutputFormat
	writer io.Writer
}

// NewOutputWriter creates a new output writer with the specified format
func NewOutputWriter(format string) *OutputWriter {
	f := OutputFormat(format)
	if f != FormatJSON && f != FormatYAML && f != FormatModern {
		f = FormatTable // default
	}
	return &OutputWriter{
		format: f,
		writer: os.Stdout,
	}
}

// SetWriter sets a custom writer (used in tests)
func (w *OutputWriter) SetWriter(writer io.Writer) {
	w.writer = writer
}

// GetFormat returns the current format
func (w *OutputWriter) GetFormat() OutputFormat {
	return w.format
}

// IsStructured returns true if the format is JSON or YAML
func (w *OutputWriter) IsStructured() bool {
	return w.format == FormatJSON || w.format == FormatYAML
}

// Write outputs the data in the configured format
func (w *OutputWriter) Write(data interface{}) error {
	switch w.format {
	case FormatJSON:
		encoder := json.NewEncoder(w.writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(data)
	case FormatYAML:
		encoder := yaml.NewEncoder(w.writer)
		defer encoder.Close()
		return encoder.Encode(data)
	default:
		// Table format is handled by individual commands
		return nil
	}
}

// WriteBytes writes raw bytes to output
func (w *OutputWriter) WriteBytes(data []byte) error {
	_, err := w.writer.Write(data)
	return err
}

// ValidFormats returns a list of valid output formats
func ValidFormats() []string {
	return []string{"table", "modern", "json", "yaml"}
}

func FormatDuration(startTime time.Time) string {
	if startTime.IsZero() {
		return ""
	}

	duration := time.Since(startTime)

	if duration.Seconds() < 60 {
		seconds := int(duration.Seconds())
		if seconds == 1 {
			return "running for 1 second"
		}
		return fmt.Sprintf("running for %d seconds", seconds)
	}

	minutes := int(duration.Minutes())
	if minutes < 60 {
		if minutes == 1 {
			return "running for 1 minute"
		}
		return fmt.Sprintf("running for %d minutes", minutes)
	}

	hours := int(duration.Hours())
	if hours == 1 {
		return "running for 1 hour"
	}
	return fmt.Sprintf("running for %d hours", hours)
}

func FormatTimestamp(ts time.Time) string {
	if ts.IsZero() {
		return "-"
	}
	return ts.Format("02/01 15:04")
}

// CopyToClipboard writes content to the clipboard as plain text.
func CopyToClipboard(clipboardContent string) error {
	return atottoclipboard.WriteAll(clipboardContent)
}

// CopyRichToClipboard copies content to the clipboard as both HTML (for rich
// text apps such as Teams/Slack) and plain text (for text editors). On
// Linux/Wayland it daemonises a clipboard server that serves both formats
// simultaneously.
func CopyRichToClipboard(html, plain string) error {
	return clipboard.WriteMultiFormat(html, plain)
}

// ShouldCopyOutput checks if the --copy flag was set on the command.
// It first checks the command's local flags, then falls back to the global flag.
func ShouldCopyOutput(cmd *cobra.Command) bool {
	// Check local flag first
	if cmd.Flags().Changed("copy") {
		copyFlag, _ := cmd.Flags().GetBool("copy")
		return copyFlag
	}
	// Fall back to global flag
	return copyToClipboardFlag
}

// OutputWithCopy handles output to terminal and optionally to clipboard.
// It ALWAYS prints to terminal, and if shouldCopy is true, also copies to clipboard.
// The clipboardContent should be formatted with markdown-style links for Teams compatibility.
func OutputWithCopy(writer io.Writer, terminalContent, clipboardContent string, shouldCopy bool) error {
	// Always print to terminal
	if _, err := fmt.Fprint(writer, terminalContent); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	// Copy to clipboard if requested
	if shouldCopy {
		if err := CopyToClipboard(clipboardContent); err != nil {
			return fmt.Errorf("failed to copy to clipboard: %w", err)
		}
		fmt.Fprintln(writer, "\n✓ Copied to clipboard!")
	}

	return nil
}

// CopyWithMessage copies content to clipboard and prints a confirmation message.
func CopyWithMessage(clipboardContent, message string) error {
	if err := CopyToClipboard(clipboardContent); err != nil {
		return err
	}
	if message != "" {
		fmt.Println(message)
	}
	return nil
}

// GenerateMarkdownLink creates a markdown-style link for Teams compatibility.
// Teams recognizes markdown links and converts them to clickable links when pasted.
func GenerateMarkdownLink(url, text string) string {
	if url == "" {
		return text
	}
	return fmt.Sprintf("[%s](%s)", text, url)
}

// GeneratePlainLink creates a plain text representation of a link.
func GeneratePlainLink(url, text string) string {
	if url == "" {
		return text
	}
	return fmt.Sprintf("%s (%s)", text, url)
}
