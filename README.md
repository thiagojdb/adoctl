# adoctl - Azure DevOps Control Tool

[![CI](https://github.com/USER/adoctl/workflows/CI/badge.svg)](https://github.com/USER/adoctl/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/USER/adoctl)](https://goreportcard.com/report/github.com/USER/adoctl)
[![codecov](https://codecov.io/gh/USER/adoctl/branch/main/graph/badge.svg)](https://codecov.io/gh/USER/adoctl)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

CLI tool for managing Azure DevOps workflows. Supports creating PRs, generating reports, monitoring builds/deployments, work item management, git hooks integration, and multi-profile configuration. Uses SQLite caching and XDG config directory standards.

## Quick Start

```bash
# Build and install
make build
sudo make install-local

# Configure
mkdir -p ~/.config/adoctl
cp config.example.yaml ~/.config/adoctl/config.yaml
# Edit config.yaml with your org/project, then:
export AZURE_PAT="your_personal_access_token"

# Try it out
adoctl repos
adoctl pr list
```

## Configuration

### Config File

Location (XDG standard): `~/.config/adoctl/config.yaml`

```yaml
azure:
  organization: "YourOrganization"
  project: "YourProject"
  # personal_access_token: ""   # Use AZURE_PAT env var instead
  api_version: "7.1"            # Optional, defaults to 7.1

threadpool:
  parallel_processes: 32        # Concurrent operations (default: 32)
```

### Environment Variables

```bash
export AZURE_ORGANIZATION="YourOrganization"
export AZURE_PROJECT="YourProject"
export AZURE_PAT="your_personal_access_token"
export ADOCTL_LOG_LEVEL="info"
export ADOCTL_THREADPOOL_SIZE="32"
```

Environment variables take priority over config file values.

**Recommendation:** Keep `organization` and `project` in the config file; use `AZURE_PAT` for the token.

## Building

```bash
# Build for current platform (outputs to dist/adoctl)
make build

# Build release binaries for all platforms (linux/darwin/windows, amd64/arm64)
make build-release

# Install to GOPATH/bin
make install

# Install to /usr/local/bin (requires sudo)
sudo make install-local

# Run in development mode
make dev ARGS="repos"

# Run tests
make test
make test-cover

# Format and lint
make fmt
make vet
make lint

# Clean build artifacts
make clean

# Show all make targets
make help
```

### Manual Build

```bash
go build -o adoctl ./cmd/adoctl

# With version info
VERSION=$(git describe --tags --always) && \
go build -ldflags "-X main.Version=$VERSION" -o adoctl ./cmd/adoctl
```

## Global Flags

All commands support these flags:

- `--timeout` - API request timeout (default: 30s)
- `--format` - Output format: `table`, `modern`, `json`, `yaml` (default: `table`)
- `--dry-run` - Show what would be done without making changes
- `--yes, -y` - Skip confirmation prompts
- `--copy` - Copy output to clipboard (markdown formatted for Teams clickable links)
- `--log-level` - Logging level: `debug`, `info`, `warn`, `error`, `fatal`, `panic` (default: `info`)

---

## PR Management

### `adoctl pr create`

Create a PR with auto-detection from git context (repo, branch, title, work items).

```bash
# Auto-detect everything from current git branch
adoctl pr create --title "My feature"

# Explicit settings
adoctl pr create --repository-name myrepo --source-branch feature --target-branch main --title "My feature"

# With reviewers and description
adoctl pr create --title "My feature" --description "Adds new functionality" --reviewers user1@domain.com

# Work items auto-extracted from branch names like feature/PBI-123
adoctl pr create --title "Fix issue"
```

**Key Flags:**
- `--repository-name` - Repository name (auto-detected from git remote)
- `--source-branch` - Source branch (auto-detected from current branch)
- `--target-branch` - Target branch (auto-detected from upstream or defaults to `main`)
- `--title` - PR title (auto-suggested from recent commit)
- `--description` - PR description
- `--reviewers` - Reviewer email(s) (repeatable)
- `--work-item-id` - Work item IDs to link (auto-extracted from branch name)
- `--no-git-context` - Disable git auto-detection

### `adoctl pr list`

```bash
# List all active PRs
adoctl pr list

# List PRs for current repository and branch
adoctl pr list --current-branch

# Filter by status
adoctl pr list --status active
adoctl pr list --status completed

# Filter by title (fuzzy match)
adoctl pr list --title-fuzzy "login"

# Filter by creator
adoctl pr list --creator self
adoctl pr list --creator "John Doe"
```

**Key Flags:**
- `--repository-name, --repo-id` - Filter by repository
- `--status` - `all`, `active`, `completed`, `abandoned` (default: `active`)
- `--target-branch, --source-branch` - Filter by branch
- `--creator` - Filter by creator (`self`, name, or ID)
- `--title-fuzzy` - Filter PR title by fuzzy match
- `--repo-fuzzy` - Filter by repository name (fuzzy match)
- `--current-branch` - Show only PRs from current git branch

### `adoctl pr status`

```bash
# Show PR status for current branch
adoctl pr status

# Show PR status for specific branch
adoctl pr status --branch feature/my-feature
```

### `adoctl pr merge`

```bash
# Merge PR
adoctl pr merge --pr 123 --repository-name myrepo

# Squash merge and delete source branch
adoctl pr merge --pr 123 --repository-name myrepo --strategy squash --delete-source

# With custom commit message
adoctl pr merge --pr 123 --repository-name myrepo --message "Merged feature"
```

**Key Flags:**
- `--strategy` - Merge strategy: `noFastForward`, `squash`, `rebase`, `rebaseMerge`
- `--delete-source` - Delete source branch after merge
- `--message` - Custom merge commit message
- `--skip-policy` - Bypass merge policy requirements

### `adoctl pr abandon`

```bash
adoctl pr abandon --pr 123 --repository-name myrepo
adoctl pr abandon --pr 123 --repository-name myrepo --comment "Needs rework"
```

### `adoctl pr approve`

```bash
adoctl pr approve --pr 123 --repository-name myrepo
```

### `adoctl pr bulk-create`

Create PRs across all repositories that have the specified source branch:

```bash
# Create PRs from feature branch to main in all repos
adoctl pr bulk-create --source-branch feature/new-feature --target-branch main --title "Release feature"

# Create PRs to multiple target branches
adoctl pr bulk-create --source-branch develop --target-branch main --target-branch release --title "Release merge"
```

### `adoctl pr link-workitems`

```bash
# Link single work item to PR
adoctl pr link-workitems --pr-id 123 --work-item-id 456

# Link multiple work items to multiple PRs
adoctl pr link-workitems --pr-id 123 --pr-id 456 --work-item-id 789 --work-item-id 101
```

### `adoctl pr pipeline`

Get PR status with CI/CD pipeline status including builds and deployments.

```bash
# Single PR (detailed format)
adoctl pr pipeline --pr 123

# Multiple PRs in modern table format
adoctl pr pipeline --pr 123 --pr 456 --format modern

# Watch mode (continuously refresh every 30s)
adoctl pr pipeline --pr 123 --watch

# Custom refresh interval
adoctl pr pipeline --pr 123 --watch --interval 10

# Read PR numbers from file
adoctl pr pipeline --file pr-list.txt

```

**Key Flags:**
- `--pr` - PR number(s) (repeatable)
- `--file` - Read PR numbers from file (one per line, ignores non-numeric lines)
- `--format` - `detailed` (default) or `modern`
- `--watch` - Watch mode: continuously refresh
- `--interval` - Refresh interval in seconds (default: 30, requires `--watch`)
- `--quiet` - Suppress sync messages
- `--cached-only` - Use cached data only

**File Format (pr-list.txt):**
```
PR's para release (from preprod):

46967
46969
46971
```

The file reader automatically extracts PR numbers, ignoring headers and empty lines.

**Detailed output example:**
```
══════════════════════════════════════════════════════════════
PR #46964: feat: some feature
══════════════════════════════════════════════════════════════
Status:    active
Commit:    41ff8e5fadfb97b390ccf2a8fda976e70f2e302b
Approvals: Approved (2/2 approved)

Build #109463
  Status:     completed
  Result:     succeeded
  Started:    2025-01-05 15:30:00
  Deployments:
    • #17363: succeeded (Approved)
```

**Modern table output example:**
```
🔀 Pull Requests (3 found)

│ PR   │ Title                │ Repository    │ Approvals      │ CI              │ CD                  │
┼──────┼──────────────────────┼───────────────┼────────────────┼─────────────────┼─────────────────────┤
│ 46964│ feat: some feature   │ my-service    │ Approved (2/2) │ ✔ Succeeded     │ ✔ Succeeded (prod)  │
│ 46965│ feat: another feature│ other-service │ Partial (1/2)  │ ⟳ Running (5m)  │ ⟳ Running (2m)      │
```

---

## Build Commands

### `adoctl build sync`

```bash
adoctl build sync          # Sync builds to local cache
adoctl build sync --force  # Force refresh
```

### `adoctl build search`

```bash
# Filter by branch and status
adoctl build search --branch main --status completed

# Filter by time range
adoctl build search --start-time-from 2024-01-01T00:00:00Z --start-time-to 2024-01-31T23:59:59Z

# Output as JSON
adoctl build search --status failed --json

# Save to file
adoctl build search --limit 10 --output builds.json --json
```

**Key Flags:** `--build-id`, `--branch`, `--repository`, `--commit`, `--status`, `--start-time-from`, `--start-time-to`, `--end-time-from`, `--end-time-to`, `--has-end-time`, `--limit`, `--output`, `--json`

---

## Deployment Commands

### `adoctl deployment sync`

```bash
adoctl deployment sync                    # Sync deployments
adoctl deployment sync --force            # Force refresh
adoctl deployment sync --release-id 123   # Sync specific release
```

### `adoctl deployment search`

```bash
adoctl deployment search --status succeeded
adoctl deployment search --repository myrepo --branch main
adoctl deployment search --release-name "Release 1.0" --limit 10
```

**Key Flags:** `--release-id`, `--release-name`, `--status`, `--repository`, `--branch`, `--start-time-from`, `--start-time-to`, `--end-time-from`, `--end-time-to`, `--artifact-date-from`, `--artifact-date-to`, `--has-end-time`, `--limit`

---

## Work Item Commands

### `adoctl workitem`

Get work item content and save to markdown with attachments.

```bash
# Single work item
adoctl workitem 12345

# Multiple work items at once
adoctl workitem 12345 12346 12347

# Save to specific directory
adoctl workitem 12345 --output /path/to/output

# Update existing work item from server
adoctl workitem 12345 --update
```

Creates a directory structure per work item:
```
workitem-12345/
├── workitem-12345.md   # Title, type, state, assignee, description, comments
└── attachments/        # All attached files
```

**Flags:** `--output, -o` (output directory), `--update, -u` (refresh from server)

---

## Repository Commands

### `adoctl repos`

```bash
adoctl repos                              # List all repositories
adoctl repos --filter-regex "^back-"     # Filter by regex
adoctl repos --filter-fuzzy "api"        # Filter by fuzzy match
adoctl repos --format json               # JSON output
```

Repositories are cached for 24 hours after first fetch.

---

## Report Commands

### `adoctl report`

Generate a PR message report, grouped by target and source branch.

```bash
# All active PRs
adoctl report --status active

# Copy to clipboard for Teams (rich markdown with clickable links)
adoctl report --status active --copy

# Filter by target branch and save to file
adoctl report --target-branch main --output pr-report.md

# Filter by creator
adoctl report --creator self

# Filter by title (regex or fuzzy)
adoctl report --title-regex ".*release.*"
adoctl report --title-fuzzy "login bug"

# Filter by linked work items
adoctl report --work-items PBI-12345 BUG-67890

# Hide requirement warnings (missing work items, merge conflicts)
adoctl report --status active --no-warnings
```

**Key Flags:**
- `--repository-name, --repo-id` - Filter by repository
- `--status` - `all`, `active`, `completed`, `abandoned`
- `--target-branch, --source-branch` - Filter by branch
- `--creator` - Filter by creator (`self`, name, or ID)
- `--title-regex, --title-fuzzy` - Filter PR title
- `--repo-regex, --repo-fuzzy` - Filter by repository name
- `--work-items` - Filter by linked work items (e.g., `PBI-12345`, `BUG-12345`)
- `--output` - Output file path (default: stdout)
- `--copy` - Copy to clipboard (HTML formatted for Teams)
- `--no-warnings` - Hide warnings for missing work items and merge conflicts

**Output format:**
```
PR's para develop (from feature-branch):
service-declaracao: PR NAME, [PR #123](https://dev.azure.com/.../pullrequest/123)
other-service: PR NAME, [PR #456](https://dev.azure.com/.../pullrequest/456)

PR's para main (from preprod):
service-empresa: PR NAME, [PR #789](https://dev.azure.com/.../pullrequest/789)
    ⚠️  Has merge conflicts
```

---

## Configuration Commands

### `adoctl config show`
```bash
adoctl config show   # Display current configuration
```

### `adoctl config path`
```bash
adoctl config path   # Show config file location
```

### `adoctl config profiles list`
```bash
adoctl config profiles list
```

### `adoctl config profiles add`
```bash
adoctl config profiles add --name work --org MyOrg --project MyProject
```

### `adoctl config profiles remove`
```bash
adoctl config profiles remove --name work
```

### `adoctl config profiles use`
```bash
adoctl config profiles use --name work
```

---

## Git Hooks Commands

### `adoctl hooks install`

```bash
adoctl hooks install                        # Install all hooks
adoctl hooks install --hook pre-push        # Install specific hook
adoctl hooks install --dir /path/to/hooks   # Custom directory
```

**Available hooks:**
- `pre-push` - Check if PR exists for current branch
- `post-commit` - Suggest creating PR after commits
- `prepare-commit-msg` - Validate commit message format

### `adoctl hooks list`
```bash
adoctl hooks list
```

### `adoctl hooks uninstall`
```bash
adoctl hooks uninstall
```

---

## Filtering

### Regex Filtering
Full Go regex syntax, case-sensitive by default:

```bash
adoctl report --title-regex "(?i)fix|bug"         # Case-insensitive
adoctl report --title-regex "^feat/.*"
adoctl repos --filter-regex "^back-"
```

### Fuzzy Matching
Subsequence matching - characters must appear in order:

```bash
# "fmt" matches "format", "formatted", "fmt: test"
adoctl pr list --title-fuzzy "fmt"

# "rfrmt" matches "reformat", "reforma tributaria"
adoctl pr list --title-fuzzy "rfrmt"
```

### Creator Filtering

```bash
adoctl report --creator self              # Your own PRs (token owner)
adoctl report --creator "Alice"          # By name (must match exactly one user)
adoctl report --creator xxxxxxxx-...     # By Azure DevOps user ID
```

If a name matches multiple users, all matches are shown:
```
Multiple creators found matching 'JohnD':
  - JohnD01 (ID: xxxxxxxx-..., PRs: 10)
  - JohnD02 (ID: yyyyyyyy-..., PRs: 7)
```

---

## Caching

adoctl uses SQLite to cache frequently accessed data.

**Cache location:**
- Linux/macOS: `~/.cache/adoctl/cache.db`
- Windows: `%LOCALAPPDATA%\adoctl\cache.db`

**Cached data:**
- Repositories: 24-hour TTL
- Users: Cached after first fetch
- Builds and deployments: Refreshed on sync

**Benefits:** Faster subsequent commands, reduced API calls, offline viewing of cached data.

---

## Git Context Auto-Detection

When run from within a git repository with an Azure DevOps remote, adoctl auto-detects:

1. **Repository** - From git remote URL
2. **Source branch** - From current git branch
3. **Target branch** - From tracked upstream or defaults to `main`
4. **PR title** - From recent commit message
5. **Work items** - From branch names like `feature/PBI-12345`

```bash
# Force git context (default behavior)
adoctl pr create --use-git-context --title "My feature"

# Disable git context
adoctl pr create --no-git-context --repository-name myrepo --source-branch feature
```

---

## Clipboard Integration

The `--copy` flag copies output to clipboard with markdown formatting compatible with Teams, Slack, and Discord.

```bash
adoctl pr list --copy
adoctl pr status --copy
adoctl report --status active --copy
adoctl pr create --title "My feature" --copy
```

- Output is also always printed to terminal
- Confirmation message shown on success
- Links are formatted as clickable in Teams

---

## Shell Completions

### Bash
```bash
echo 'source <(adoctl completion bash)' >> ~/.bashrc
```

### Zsh
```bash
make install-completion-zsh
# Requires compinit in ~/.zshrc:
echo "autoload -U compinit && compinit" >> ~/.zshrc
exec zsh
```

### Fish
```bash
adoctl completion fish > ~/.config/fish/completions/adoctl.fish
```

### PowerShell
```powershell
adoctl completion powershell >> $PROFILE
```

**Completion features:** Repository names (cached), branch names, PR statuses, output formats, creator names, work item type prefixes.

---

## Common Workflows

### Creating a PR from Current Branch
```bash
# From branch feature/PBI-12345
adoctl pr create --title "Implement new feature"
# Auto-detects: repo, source branch, target branch (main)
# Auto-extracts work item: PBI-12345
```

### Monitoring Multiple PRs
```bash
# Create pr-list.txt with PR numbers, then:
adoctl pr pipeline --file pr-list.txt --watch --format modern
```

### Generating a Release Report
```bash
adoctl report --status active --target-branch main --copy
# Paste into Teams with proper formatting and clickable links
```

### Bulk PR Creation
```bash
adoctl pr bulk-create \
  --source-branch release/v1.0 \
  --target-branch main \
  --target-branch develop \
  --title "Release v1.0"
```

### Switching Between Organizations
```bash
adoctl config profiles add --name client --org ClientOrg --project ClientProject
adoctl config profiles use --name client
adoctl pr list
```

---

## Error Reference

| Error | Solution |
|-------|----------|
| "repository not found" | Check repository name or use `--use-git-context` |
| "could not determine current branch" | Run from within a git repository |
| "PR title is required" | Use `--title` or ensure recent commits exist |
| "token not configured" | Set `AZURE_PAT` environment variable |
| "operation cancelled by user" | Use `--yes` to skip confirmation prompts |
| "multiple creators found" | Use a more specific name or the exact user ID |
