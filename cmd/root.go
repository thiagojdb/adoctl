package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"adoctl/pkg/completions"
	"adoctl/pkg/errors"
	"adoctl/pkg/logger"

	"github.com/spf13/cobra"
)

const (
	unknownValue = "unknown"
)

var (
	Version   string
	BuildTime string
	GitCommit string
)

var defaultTimeout = 30 * time.Second
var globalTimeout time.Duration
var outputFormat string
var dryRunFlag bool
var assumeYesFlag bool
var copyToClipboardFlag bool
var logLevel string

var rootCmd = &cobra.Command{
	Use:   "adoctl",
	Short: "Azure DevOps Control Tool",
	Long: `CLI tool for managing Azure DevOps workflows. Supports creating PRs,
generating reports, monitoring builds/deployments, and work item management.
Uses SQLite caching and XDG config directory for configuration.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if globalTimeout <= 0 {
			globalTimeout = defaultTimeout
		}
		// Set log level: explicit flag takes precedence over env var
		level := logLevel
		if !cmd.Flags().Changed("log-level") {
			if envLevel := os.Getenv("ADOCTL_LOG_LEVEL"); envLevel != "" {
				level = envLevel
			}
		}
		logger.SetLevel(level)
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		ver := Version
		if ver == "" {
			ver = "dev"
		}
		bt := BuildTime
		if bt == "" {
			bt = unknownValue
		}
		gc := GitCommit
		if gc == "" {
			gc = unknownValue
		}

		fmt.Printf("adoctl version %s\n", ver)
		fmt.Printf("Built: %s\n", bt)
		fmt.Printf("Git commit: %s\n", gc)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		exitCode := errors.HandleReturn(err)
		os.Exit(int(exitCode))
	}
}

func GetContext() (context.Context, context.CancelFunc) {
	timeout := globalTimeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return context.WithTimeout(context.Background(), timeout)
}

func init() {
	RegisterCommands(rootCmd)

	rootCmd.PersistentFlags().DurationVar(&globalTimeout, "timeout", defaultTimeout, "Timeout for API requests (e.g., 30s, 1m)")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "format", "table", "Output format (table, modern, json, yaml)")
	rootCmd.PersistentFlags().BoolVar(&dryRunFlag, "dry-run", false, "Show what would be done without making changes")
	rootCmd.PersistentFlags().BoolVarP(&assumeYesFlag, "yes", "y", false, "Skip confirmation prompts")
	rootCmd.PersistentFlags().BoolVar(&copyToClipboardFlag, "copy", false, "Copy output to clipboard (supports rich HTML for Teams)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error, fatal, panic)")

	completions.RegisterCompletions(rootCmd)
}
