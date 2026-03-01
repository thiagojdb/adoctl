# adoctl - Azure DevOps Control Tool

CLI tool for managing Azure DevOps workflows including PRs, builds, deployments, and work items.

## Overview

**adoctl** is a Go-based CLI tool that provides comprehensive management of Azure DevOps resources:

- Pull Request (PR) lifecycle management
- Build and deployment monitoring
- Work item content extraction
- Repository management
- Git hooks integration
- Multi-profile configuration support

## Configuration

### Config File Location
`~/.config/adoctl/config.yaml` (XDG standard)

### Environment Variables (higher priority than config file)
```bash
export AZURE_ORGANIZATION="YourOrganization"
export AZURE_PROJECT="YourProject"
export AZURE_PAT="your_personal_access_token"
```

### Priority Order
1. Environment variables (highest)
2. Config file values
3. Default values (lowest)

**Recommendation:** Keep organization/project in config file, use `AZURE_PAT` environment variable for the token.

## Command Reference

### Global Flags
All commands support these flags:
- `--timeout` - Timeout for API requests (default: 30s)
- `--format` - Output format: table, modern, json, yaml (default: table)
- `--dry-run` - Show what would be done without making changes
- `--yes, -y` - Skip confirmation prompts
- `--copy` - Copy output to clipboard (supports markdown for Teams clickable links)

---

## PR Management Commands

### `adoctl pr create`
Create a new pull request with auto-detection from git context.

**Usage:**
```bash
# Create PR with auto-detection (when in git repo)
adoctl pr create --title "My feature"

# Create PR with explicit settings
adoctl pr create --repository-name myrepo --source-branch feature --target-branch main --title "My feature"

# Create PR with reviewers and description
adoctl pr create --title "My feature" --description "Adds new functionality" --reviewers user1@domain.com

# Create PR and link work items (auto-extracted from branch like feature/PBI-123)
adoctl pr create --title "Fix issue"
```

**Key Flags:**
- `--repository-name` - Repository name (auto-detected from git)
- `--repo-id` - Repository ID (alternative to --repository-name)
- `--source-branch` - Source branch (auto-detected from current git branch)
- `--target-branch` - Target branch (auto-detected from upstream or defaults to 'main')
- `--title` - PR title (auto-suggested from recent commit)
- `--description` - PR description
- `--reviewers` - List of reviewer IDs (can specify multiple)
- `--work-item-id` - Work item IDs to link (auto-extracted from branch)
- `--use-git-context` - Use git context for auto-detection (default: true)
- `--no-git-context` - Disable git context auto-detection

### `adoctl pr list`
List pull requests with optional filtering.

**Usage:**
```bash
# List all PRs
adoctl pr list

# List PRs for current repository
adoctl pr list --use-git-context

# List PRs for current branch only
adoctl pr list --current-branch

# List active PRs with fuzzy title search
adoctl pr list --status active --title-fuzzy "login"

# Filter by creator
adoctl pr list --creator self
adoctl pr list --creator "John Doe"
```

**Key Flags:**
- `--repository-name, --repo-id` - Filter by repository
- `--status` - PR status filter: all, active, completed, abandoned
- `--target-branch, --source-branch` - Filter by branch
- `--creator` - Filter by creator (use 'self', name, or ID)
- `--title-fuzzy` - Filter PR title by fuzzy match
- `--repo-fuzzy` - Filter by repository name (fuzzy match)
- `--current-branch` - Show only PRs from current git branch

### `adoctl pr status`
Show PR status for current branch.

**Usage:**
```bash
# Show PR status for current branch
adoctl pr status

# Show PR status for specific branch
adoctl pr status --branch feature/my-feature
```

### `adoctl pr merge`
Merge a pull request.

**Usage:**
```bash
# Merge PR #123
adoctl pr merge --pr 123 --repo-id <repo-id>

# Merge with squash strategy and delete source branch
adoctl pr merge --pr 123 --repo-id <repo-id> --strategy squash --delete-source

# Merge with custom commit message
adoctl pr merge --pr 123 --repo-id <repo-id> --message "Merged featureXYZ"
```

**Key Flags:**
- `--pr` (required) - Pull request ID to merge
- `--strategy` - Merge strategy: noFastForward, squash, rebase, rebaseMerge
- `--delete-source` - Delete source branch after merge
- `--message` - Custom merge commit message
- `--skip-policy` - Bypass merge policy requirements

### `adoctl pr abandon`
Abandon a pull request.

**Usage:**
```bash
# Abandon PR #123
adoctl pr abandon --pr 123 --repo-id <repo-id>

# Abandon with a comment
adoctl pr abandon --pr 123 --repo-id <repo-id> --comment "Abandoning - needs rework"
```

### `adoctl pr bulk-create`
Create PRs across all repositories that have the specified source branch.

**Usage:**
```bash
# Create PRs from feature branch to main in all repos
adoctl pr bulk-create --source-branch feature --target-branch main --title "Merge feature"

# Create PRs to multiple target branches
adoctl pr bulk-create --source-branch develop --target-branch main --target-branch release --title "Release merge"
```

**Key Flags:**
- `--source-branch` (required) - Source branch name
- `--target-branch` (required, can specify multiple) - Target branch name(s)
- `--title` (required) - PR title
- `--work-item-id` - Work item IDs to link

### `adoctl pr link-workitems`
Link work items to existing pull requests.

**Usage:**
```bash
# Link single work item to PR
adoctl pr link-workitems --pr-id 123 --work-item-id 456

# Link multiple work items to multiple PRs
adoctl pr link-workitems --pr-id 123 --pr-id 456 --work-item-id 789 --work-item-id 101
```

### `adoctl pr pipeline`
Get PR status and CI/CD pipeline status including builds and deployments. Also accessible as `pr pipeline-status`.

**Usage:**
```bash
# Get pipeline status for single PR (detailed format)
adoctl pr pipeline --pr 123

# Get status for multiple PRs in modern table format
adoctl pr pipeline --pr 123 --pr 456 --format modern

# Watch PR status with refresh
adoctl pr pipeline --pr 123 --watch

# Watch with custom interval (seconds)
adoctl pr pipeline --pr 123 --watch --interval 10

# Read PR numbers from file
adoctl pr pipeline --file pr-list.txt

```

**Key Flags:**
- `--pr` - PR number(s) (can specify multiple)
- `--file` - Read PR numbers from file
- `--format` - Output format: 'detailed' or 'modern'
- `--watch` - Watch mode: continuously refresh
- `--interval` - Refresh interval in seconds (default: 30)
- `--quiet` - Suppress sync messages
- `--cached-only` - Use cached data only

### `adoctl pr approve`
Approve a pull request.

**Usage:**
```bash
adoctl pr approve --pr 123 --repo-id <repo-id>
```

---

## Build Commands

### `adoctl build sync`
Sync builds from Azure DevOps to local cache.

**Usage:**
```bash
# Sync builds to local cache
adoctl build sync

# Force sync all builds (ignore cache)
adoctl build sync --force
```

### `adoctl build search`
Search builds in local cache with filters.

**Usage:**
```bash
# Search builds by branch
adoctl build search --branch main

# Search builds by status
adoctl build search --status completed

# Search with time filters
adoctl build search --start-time-from 2024-01-01T00:00:00Z --start-time-to 2024-01-31T23:59:59Z

# Search and output as JSON
adoctl build search --status failed --json

# Limit results and save to file
adoctl build search --limit 10 --output builds.json --json
```

**Key Flags:**
- `--build-id` - Filter by build ID
- `--branch` - Filter by branch name
- `--repository` - Filter by repository/pipeline name
- `--commit` - Filter by commit/build number
- `--status` - Filter by build status
- `--start-time-from, --start-time-to` - Filter by start time (RFC3339 format)
- `--end-time-from, --end-time-to` - Filter by end time (RFC3339 format)
- `--has-end-time` - Filter by end time existence (true/false)
- `--limit` - Limit number of results
- `--output` - Output file path
- `--json` - Output in JSON format

---

## Deployment Commands

### `adoctl deployment sync`
Sync deployments from Azure DevOps to local cache.

**Usage:**
```bash
# Sync deployments to local cache
adoctl deployment sync

# Force sync all deployments
adoctl deployment sync --force

# Sync deployments for specific release
adoctl deployment sync --release-id 123
```

### `adoctl deployment search`
Search deployments in local cache with filters.

**Usage:**
```bash
# Search deployments by status
adoctl deployment search --status succeeded

# Search deployments by repository
adoctl deployment search --repository myrepo

# Search with time filters
adoctl deployment search --start-time-from 2024-01-01T00:00:00Z

# Limit results
adoctl deployment search --limit 10

# Filter by release name
adoctl deployment search --release-name "Release 1.0"
```

**Key Flags:**
- `--release-id` - Filter by release ID
- `--release-name` - Filter by release name
- `--status` - Filter by deployment status
- `--repository` - Filter by repository/pipeline name
- `--branch` - Filter by branch name
- `--start-time-from, --start-time-to` - Filter by start time (RFC3339 format)
- `--end-time-from, --end-time-to` - Filter by end time (RFC3339 format)
- `--artifact-date-from, --artifact-date-to` - Filter by artifact date
- `--has-end-time` - Filter by end time existence (true/false)
- `--limit` - Limit number of results

---

## Work Item Commands

### `adoctl workitem`
Get work item content and save to markdown.

**Usage:**
```bash
# Get single work item
adoctl workitem 123

# Get multiple work items
adoctl workitem 123 456 789

# Get work item and save to specific directory
adoctl workitem 123 --output ./work-items

# Update existing work item from server
adoctl workitem 123 --update --output ./work-items
```

**Creates directory structure:**
```
workitem-12345/
├── workitem-12345.md     # Markdown file with work item details
└── attachments/          # Directory with all attached files
```

**Key Flags:**
- `--output, -o` - Output directory for work item files
- `--update, -u` - Update existing work item from server (replaces saved content)

---

## Repository Commands

### `adoctl repos`
List all repositories in the Azure DevOps project.

**Usage:**
```bash
# List all repositories
adoctl repos

# Filter by regex pattern
adoctl repos --filter-regex "frontend.*|backend.*"

# Filter by fuzzy match
adoctl repos --filter-fuzzy "api"

# Output as JSON
adoctl repos --format json

# Output as YAML
adoctl repos --format yaml
```

**Key Flags:**
- `--filter-regex` - Filter repository name by regex pattern
- `--filter-fuzzy` - Filter repository name by fuzzy match

---

## Report Commands

### `adoctl report`
Generate PR message report with filtering and output options.

**Usage:**
```bash
# Generate report for all active PRs
adoctl report --status active

# Generate report for specific repository
adoctl report --repository-name myrepo

# Generate and copy to clipboard for Teams
adoctl report --status active --copy

# Filter by target branch and save to file
adoctl report --target-branch main --output pr-report.md

# Filter by creator
adoctl report --creator self

# Use regex pattern for title filter
adoctl report --title-regex ".*release.*"

# Use fuzzy match for title
adoctl report --title-fuzzy "login bug"

# Hide requirement warnings
adoctl report --status active --no-warnings

# Filter by work items
adoctl report --work-items PBI-12345 BUG-67890
```

**Key Flags:**
- `--repository-name, --repo-id` - Filter by repository
- `--status` - PR status filter: all, active, completed, abandoned
- `--target-branch, --source-branch` - Filter by branch
- `--creator` - Filter by creator (use 'self', name, or ID)
- `--title-regex, --title-fuzzy` - Filter PR title
- `--repo-regex, --repo-fuzzy` - Filter by repository name
- `--work-items` - Filter by linked work items (e.g., PBI-12345, BUG-12345)
- `--output` - Output file path (default: stdout)
- `--copy` - Copy report to clipboard for Teams (HTML formatted)
- `--no-warnings` - Hide requirement warnings (work items, merge conflicts)

---

## Configuration Commands

### `adoctl config show`
Show current configuration.

**Usage:**
```bash
adoctl config show
```

### `adoctl config path`
Show configuration file path.

**Usage:**
```bash
adoctl config path
```

### `adoctl config profiles list`
List all configuration profiles.

**Usage:**
```bash
adoctl config profiles list
```

### `adoctl config profiles add`
Add a new configuration profile.

**Usage:**
```bash
# Add profile interactively
adoctl config profiles add --name work --org MyOrg --project MyProject

# Add profile with token (prefer env var instead)
adoctl config profiles add --name work --org MyOrg --project MyProject --token $AZURE_PAT
```

### `adoctl config profiles remove`
Remove a configuration profile.

**Usage:**
```bash
adoctl config profiles remove --name work
```

### `adoctl config profiles use`
Switch to a profile.

**Usage:**
```bash
adoctl config profiles use --name work
```

---

## Git Hooks Commands

### `adoctl hooks install`
Install git hooks for adoctl integration.

**Usage:**
```bash
# Install all hooks
adoctl hooks install

# Install specific hook
adoctl hooks install --hook pre-push

# Install to custom directory
adoctl hooks install --dir /path/to/hooks
```

**Available hooks:**
- `pre-push` - Check if PR exists for current branch
- `post-commit` - Suggest creating PR after commits
- `prepare-commit-msg` - Validate PR title format

### `adoctl hooks list`
List installed adoctl hooks.

**Usage:**
```bash
adoctl hooks list
```

### `adoctl hooks uninstall`
Uninstall adoctl git hooks.

**Usage:**
```bash
adoctl hooks uninstall
```

---

## Utility Commands

### `adoctl version`
Show version information.

**Usage:**
```bash
adoctl version
```

---

## Filtering Patterns

### Regex Filtering
Uses Go regex syntax (case-sensitive by default):
```bash
# Case-insensitive regex
adoctl report --title-regex "(?i)fix|bug"

# Match patterns
adoctl report --title-regex "^feat/.*"
adoctl repos --filter-regex "^back-"
```

### Fuzzy Matching
Subsequence matching - matches if characters appear in order:
```bash
# "fmt" matches "format", "formatted", "fmt: test"
adoctl pr list --title-fuzzy "fmt"

# "rfrmt" matches "reformat", "reforma tributaria"
adoctl pr list --title-fuzzy "rfrmt"
```

---

## Caching

adoctl uses SQLite for caching frequently accessed data:

**Cached Data:**
- Repositories: Full list with metadata (24 hour TTL)
- Users: Creator/user information
- Builds: Build information
- Deployments: Deployment information

**Cache Location:**
- Linux/macOS: `~/.cache/adoctl/cache.db`
- Windows: `%LOCALAPPDATA%\adoctl\cache.db`

**Cache Benefits:**
- Faster subsequent commands
- Reduced API rate limit usage
- Offline viewing of cached data

---

## Git Context Auto-Detection

When run from within a git repository with an Azure DevOps remote, adoctl can auto-detect:

1. **Repository** - From git remote URL
2. **Source Branch** - From current git branch
3. **Target Branch** - From tracked upstream or default branch
4. **PR Title** - From recent commit message
5. **Work Items** - From branch names like `feature/PBI-12345`

**Control auto-detection:**
```bash
# Force git context usage (default)
adoctl pr create --use-git-context --title "My feature"

# Disable git context
adoctl pr create --no-git-context --repository-name myrepo --source-branch feature
```

---

## Common Workflows

### Creating a PR from Current Branch
```bash
# From within a git repo on branch feature/PBI-12345
adoctl pr create --title "Implement new feature"
# Auto-detects: repo, source branch (feature/PBI-12345), target branch (main)
# Auto-extracts work item: PBI-12345
```

### Monitoring Multiple PRs
```bash
# Create a file with PR numbers
cat > prs.txt <<EOF
123
456
789
EOF

# Watch all PRs with modern format
adoctl pr pipeline --file prs.txt --watch --format modern
```

### Generating Release Report
```bash
# Get all active PRs targeting main
adoctl report --status active --target-branch main --copy
# Paste into Teams with proper formatting
```

### Bulk PR Creation
```bash
# Create PRs from release branch to main and develop
adoctl pr bulk-create \
  --source-branch release/v1.0 \
  --target-branch main \
  --target-branch develop \
  --title "Release v1.0"
```

---

## Best Practices

1. **Use git context** for faster PR creation when in git repositories
2. **Use environment variables** for sensitive data (AZURE_PAT)
3. **Use fuzzy matching** for quick searches when exact spelling is unknown
4. **Use regex** for precise pattern matching
5. **Use watch mode** for monitoring builds/deployments
7. **Create profiles** for working with multiple Azure DevOps organizations
8. **Use dry-run** to preview destructive operations
9. **Cache frequently** - most operations benefit from caching

---

## Error Handling

Common errors and solutions:

| Error | Solution |
|-------|----------|
| "repository not found" | Check repository name or use --use-git-context |
| "could not determine current branch" | Run from within a git repository |
| "PR title is required" | Use --title or ensure recent commits exist |
| "token not configured" | Set AZURE_PAT environment variable |
| "operation cancelled by user" | Use --yes to skip confirmation prompts |

---

## Clipboard Integration

The `--copy` flag allows you to copy command output to the clipboard with markdown formatting for easy sharing in Teams, Slack, or other chat platforms.

### Supported Commands

All commands that produce text output support `--copy`:

| Command | Copied Content |
|---------|---------------|
| `adoctl pr list` | Formatted PR list with clickable links |
| `adoctl pr status` | PR status with links |
| `adoctl pr create` | Created PR details with link |
| `adoctl pr pipeline` | Pipeline status summary |
| `adoctl report` | Full report (markdown formatted) |
| `adoctl repos` | Repository list with links |
| `adoctl workitem` | Work item summary |
| `adoctl build search` | Build list |
| `adoctl deployment search` | Deployment list |

### Usage Examples

```bash
# Copy PR list to clipboard
adoctl pr list --copy

# Copy specific PR status
adoctl pr status --copy

# Copy report for sharing in Teams
adoctl report --status active --copy

# Copy after creating a PR
adoctl pr create --title "My feature" --copy
```

### Teams Compatibility

The copied content uses markdown formatting that Microsoft Teams recognizes:

- **Clickable links**: `[PR #123: Title](https://dev.azure.com/...)`
- **Bold text**: `**Status:** Active`
- **Code formatting**: `` `branch-name` ``
- **Bullet lists**: `- Item 1`

When pasted into Teams, these render as rich text with clickable links.

### Clipboard Features

- **Always prints to terminal**: Output is displayed even when copying
- **Confirmation message**: Shows "✓ Copied to clipboard!" on success
- **Single flag**: Works across all commands
- **Markdown formatted**: Compatible with Teams, Slack, Discord, etc.

---

## Shell Completions

Enable tab completions for faster command entry:

**Bash:**
```bash
echo 'source <(adoctl completion bash)' >> ~/.bashrc
```

**Zsh:**
```bash
make install-completion-zsh
```

**Fish:**
```bash
adoctl completion fish > ~/.config/fish/completions/adoctl.fish
```

**Completion features:**
- Repository names
- Branch names
- PR statuses
- Output formats
- Creator names
- Work item types
