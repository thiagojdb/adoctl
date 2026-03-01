package cmd

import (
	"encoding/json"
	"os"

	"adoctl/pkg/clipboard"

	"github.com/spf13/cobra"
)

var clipboardServeCmd = &cobra.Command{
	Use:    "__clipboard-serve",
	Hidden: true,
	Short:  "Internal: serve clipboard content over Wayland (do not call directly)",
	RunE: func(cmd *cobra.Command, args []string) error {
		var payload struct{ HTML, Plain string }
		if err := json.NewDecoder(os.Stdin).Decode(&payload); err != nil {
			return err
		}
		return clipboard.ServeClipboard(payload.HTML, payload.Plain)
	},
}
