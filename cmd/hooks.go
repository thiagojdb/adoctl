package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"adoctl/pkg/git"

	"github.com/spf13/cobra"
)

var (
	hooksDir string
)

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage git hooks integration",
	Long: `Install and manage git hooks for adoctl integration.

Git hooks allow adoctl to integrate with your git workflow, providing
automated checks and suggestions during common git operations.`,
	Example: `  # Install all hooks
  adoctl hooks install

  # Install specific hook
  adoctl hooks install --hook pre-push

  # Install to custom directory
  adoctl hooks install --dir /path/to/hooks

  # List installed hooks
  adoctl hooks list

  # Uninstall hooks
  adoctl hooks uninstall`,
}

var hooksInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install git hooks",
	Long:  `Install adoctl git hooks into your repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := hooksDir
		if targetDir == "" {
			// Try to find the .git/hooks directory
			gitDir, err := git.GetGitDir()
			if err != nil {
				return fmt.Errorf("failed to find git directory: %w\nUse --dir to specify hooks directory", err)
			}
			targetDir = filepath.Join(gitDir, "hooks")
		}

		if _, err := os.Stat(targetDir); os.IsNotExist(err) {
			return fmt.Errorf("hooks directory does not exist: %s", targetDir)
		}

		hookType, _ := cmd.Flags().GetString("hook")
		hooksToInstall := []string{"pre-push", "post-commit", "prepare-commit-msg"}
		if hookType != "" {
			hooksToInstall = []string{hookType}
		}

		installed := 0
		for _, hook := range hooksToInstall {
			hookPath := filepath.Join(targetDir, hook)

			// Check if hook already exists
			if _, err := os.Stat(hookPath); err == nil {
				// Check if it's our hook
				content, err := os.ReadFile(hookPath)
				if err != nil {
					fmt.Printf("Warning: Could not read existing %s hook: %v\n", hook, err)
					continue
				}
				if !isAdoctlHook(string(content)) {
					fmt.Printf("Skipping %s: existing hook found (not managed by adoctl)\n", hook)
					continue
				}
			}

			if err := installHook(hookPath, hook); err != nil {
				fmt.Printf("Failed to install %s: %v\n", hook, err)
				continue
			}
			fmt.Printf("Installed %s hook\n", hook)
			installed++
		}

		fmt.Printf("\nInstalled %d hook(s) to %s\n", installed, targetDir)
		if installed > 0 {
			fmt.Println("\nHook descriptions:")
			fmt.Println("  pre-push:        Check if PR exists for current branch")
			fmt.Println("  post-commit:     Suggest creating PR after commits")
			fmt.Println("  prepare-commit-msg: Validate PR title format")
		}

		return nil
	},
}

var hooksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed adoctl hooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := hooksDir
		if targetDir == "" {
			gitDir, err := git.GetGitDir()
			if err != nil {
				return fmt.Errorf("failed to find git directory: %w", err)
			}
			targetDir = filepath.Join(gitDir, "hooks")
		}

		hooks := []string{"pre-push", "post-commit", "prepare-commit-msg"}
		fmt.Println("Installed hooks:")
		found := false
		for _, hook := range hooks {
			hookPath := filepath.Join(targetDir, hook)
			status := "not installed"

			if content, err := os.ReadFile(hookPath); err == nil {
				if isAdoctlHook(string(content)) {
					status = "installed (adoctl)"
					found = true
				} else {
					status = "other hook exists"
				}
			}

			fmt.Printf("  %s: %s\n", hook, status)
		}

		if !found {
			fmt.Println("\nNo adoctl hooks are currently installed.")
			fmt.Println("Run 'adoctl hooks install' to install them.")
		}

		return nil
	},
}

var hooksUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall adoctl git hooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := hooksDir
		if targetDir == "" {
			gitDir, err := git.GetGitDir()
			if err != nil {
				return fmt.Errorf("failed to find git directory: %w", err)
			}
			targetDir = filepath.Join(gitDir, "hooks")
		}

		hooks := []string{"pre-push", "post-commit", "prepare-commit-msg"}
		removed := 0

		for _, hook := range hooks {
			hookPath := filepath.Join(targetDir, hook)
			content, err := os.ReadFile(hookPath)
			if err != nil {
				continue
			}

			if !isAdoctlHook(string(content)) {
				fmt.Printf("Skipping %s: not managed by adoctl\n", hook)
				continue
			}

			if err := os.Remove(hookPath); err != nil {
				fmt.Printf("Failed to remove %s: %v\n", hook, err)
				continue
			}

			fmt.Printf("Removed %s hook\n", hook)
			removed++
		}

		fmt.Printf("\nRemoved %d hook(s)\n", removed)
		return nil
	},
}

// isAdoctlHook checks if a hook script was created by adoctl
func isAdoctlHook(content string) bool {
	return len(content) > 0 && contains(content, "# adoctl hook")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func installHook(hookPath, hookType string) error {
	var script string

	switch hookType {
	case "pre-push":
		script = prePushHook
	case "post-commit":
		script = postCommitHook
	case "prepare-commit-msg":
		script = prepareCommitMsgHook
	default:
		return fmt.Errorf("unknown hook type: %s", hookType)
	}

	return os.WriteFile(hookPath, []byte(script), 0600)
}

const prePushHook = `#!/bin/sh
# adoctl hook - pre-push
# Checks if a PR exists for the current branch before pushing

current_branch=$(git rev-parse --abbrev-ref HEAD)

# Skip check for main/master branches
if [ "$current_branch" = "main" ] || [ "$current_branch" = "master" ]; then
    exit 0
fi

# Check if adoctl is available
if ! command -v adoctl > /dev/null 2>&1; then
    exit 0
fi

# Check for PR
if adoctl pr list --status active --format json 2>/dev/null | grep -q "$current_branch"; then
    exit 0
fi

echo ""
echo "⚠️  Warning: No active PR found for branch '$current_branch'"
echo ""
echo "You may want to create a PR with:"
echo "  adoctl pr create --source-branch $current_branch"
echo ""

# Allow the push to proceed
exit 0
`

const postCommitHook = `#!/bin/sh
# adoctl hook - post-commit
# Suggests creating a PR after multiple commits

# Count commits since branch creation or last push
commit_count=$(git rev-list --count HEAD ^@{u} 2>/dev/null || echo "0")

# Only suggest after 3+ commits
if [ "$commit_count" -ge 3 ]; then
    current_branch=$(git rev-parse --abbrev-ref HEAD)

    # Skip for main/master
    if [ "$current_branch" = "main" ] || [ "$current_branch" = "master" ]; then
        exit 0
    fi

    echo ""
    echo "💡 Tip: You have $commit_count commits on '$current_branch'"
    echo "Consider creating a PR: adoctl pr create"
    echo ""
fi

exit 0
`

const prepareCommitMsgHook = `#!/bin/sh
# adoctl hook - prepare-commit-msg
# Validates PR title format if this looks like a PR description

COMMIT_MSG_FILE=$1
COMMIT_SOURCE=$2

# Only validate regular commits (not merge, squash, etc.)
if [ -n "$COMMIT_SOURCE" ] && [ "$COMMIT_SOURCE" != "message" ]; then
    exit 0
fi

# Check if commit message follows conventional commits
grep -qE "^(feat|fix|docs|style|refactor|test|chore|ci|build|perf)(\(.+\))?: .+" "$COMMIT_MSG_FILE"
if [ $? -ne 0 ]; then
    # Not enforcing, just a warning
    : # No action, just informational
fi

exit 0
`

func init() {
	hooksCmd.AddCommand(hooksInstallCmd)
	hooksCmd.AddCommand(hooksListCmd)
	hooksCmd.AddCommand(hooksUninstallCmd)

	hooksInstallCmd.Flags().StringVar(&hooksDir, "dir", "", "Custom hooks directory (default: .git/hooks)")
	hooksInstallCmd.Flags().String("hook", "", "Install specific hook only (pre-push, post-commit, prepare-commit-msg)")

	hooksListCmd.Flags().StringVar(&hooksDir, "dir", "", "Custom hooks directory")
	hooksUninstallCmd.Flags().StringVar(&hooksDir, "dir", "", "Custom hooks directory")
}
