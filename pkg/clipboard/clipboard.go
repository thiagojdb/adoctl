// Package clipboard provides clipboard access with dual-format support.
// On Linux/Wayland it daemonizes a clipboard server that serves both
// text/html and text/plain simultaneously, so pasting into rich-text apps
// (Teams, Slack) renders links while pasting into plain-text editors yields
// clean text without Markdown syntax.
package clipboard
