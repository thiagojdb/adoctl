package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

const (
	responseYes = "yes"
	responseY   = "y"
)

// IsDryRun returns true if dry-run mode is enabled
func IsDryRun() bool {
	return dryRunFlag
}

// IsAssumeYes returns true if we should skip confirmation prompts
func IsAssumeYes() bool {
	return assumeYesFlag
}

// PrintDryRun prints a message indicating what would happen in dry-run mode
func PrintDryRun(format string, args ...interface{}) {
	yellow := color.New(color.FgYellow, color.Bold)
	_, _ = yellow.Print("[DRY-RUN] ")
	fmt.Printf(format+"\n", args...)
}

// PrintDryRunAction prints a dry-run action with details
func PrintDryRunAction(action string, details map[string]string) {
	yellow := color.New(color.FgYellow, color.Bold)
	cyan := color.New(color.FgCyan)

	_, _ = yellow.Printf("[DRY-RUN] Would %s:\n", action)
	for key, value := range details {
		_, _ = cyan.Printf("  %s: ", key)
		fmt.Println(value)
	}
}

// ConfirmPrompt asks the user for confirmation
func ConfirmPrompt(message string) (bool, error) {
	if assumeYesFlag {
		return true, nil
	}

	yellow := color.New(color.FgYellow)
	_, _ = yellow.Printf("%s [y/N]: ", message)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == responseY || response == responseYes, nil
}

// ConfirmDestructive prompts for confirmation before a destructive action
func ConfirmDestructive(action string, details map[string]string) (bool, error) {
	if dryRunFlag {
		PrintDryRunAction(action, details)
		return false, nil // Return false to indicate we didn't actually do it
	}

	red := color.New(color.FgRed, color.Bold)
	_, _ = red.Printf("Warning: You are about to %s\n\n", action)

	if len(details) > 0 {
		for key, value := range details {
			fmt.Printf("  %s: %s\n", key, value)
		}
		fmt.Println()
	}

	return ConfirmPrompt("Do you want to continue")
}

// RequireConfirmation exits if confirmation is denied
func RequireConfirmation(action string, details map[string]string) error {
	confirmed, err := ConfirmDestructive(action, details)
	if err != nil {
		return err
	}
	if !confirmed {
		return fmt.Errorf("operation canceled by user")
	}
	return nil
}

func init() {
	// These will be bound to root command flags
}
