# Development Guide

## Project Structure

```
adoctl/
 ├── cmd/
 │   └── adoctl/               # CLI application entry point
 │       └── main.go
 ├── pkg/
 │   ├── config/                # Configuration management
 │   │   └── config.go
 │   └── prmessagegen/         # Core library (can be imported)
 │       ├── azure_devops_client.go  # Azure DevOps API client
 │       ├── cache.go                # SQLite cache management
 │       ├── pr_manager.go           # PR business logic
 │       ├── interactive.go          # Interactive CLI prompts
 │       └── main.go                 # Command handlers
 ├── dist/                   # Build output (not in git)
 ├── Makefile               # Build automation
 ├── go.mod
 ├── go.sum
 └── README.md
```

## Development Workflow

### 1. Make Changes

Edit files in `cmd/adoctl/` or `pkg/prmessagegen/`

### 2. Test Locally

```bash
# Run directly (no build needed)
make dev ARGS="repos"

# Build and test binary
make build
./dist/adoctl version

# Run tests
make test

# Run tests with coverage
make test-cover
```

### 3. Code Quality

```bash
# Format code
make fmt

# Run go vet
make vet

# Run linter (requires golangci-lint)
make lint
```

### Testing Guidelines

#### Running Tests

```bash
# Run all unit tests
make test

# Run tests with race detection
go test -race ./...

# Run tests with coverage
make test-cover

# Run integration tests (requires AZURE_PAT)
go test -tags=integration ./pkg/integration/... -v
```

#### Writing Tests

- **Unit tests**: Test individual functions and methods in isolation
- **Integration tests**: Test interactions with external services (Azure DevOps API)
- Use the `-tags=integration` build tag for tests that require external dependencies
- Mock external dependencies using `httptest.Server` for Azure API tests
- Aim for 70%+ code coverage

#### Test Organization

- Place test files in the same package as the code being tested
- Name test files with `_test.go` suffix
- Use table-driven tests for testing multiple scenarios
- Use subtests with `t.Run()` for better organization and reporting

Example:
```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"empty string", "", ""},
        {"normal case", "hello", "HELLO"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := MyFunction(tt.input)
            if result != tt.expected {
                t.Errorf("got %q, want %q", result, tt.expected)
            }
        })
    }
}
```

### 4. Build Release

```bash
# Build for current platform
make build

# Build for all platforms
make build-release
```

### 5. Install Locally

```bash
# Install to /usr/local/bin
sudo make install-local

# Test installation
adoctl version
```

## Adding a New Command

1. Add command handler in `pkg/prmessagegen/main.go`:

```go
func NewCommand(args []string) {
    flags := flag.NewFlagSet("newcmd", flag.ExitOnError)
    // Add flags...
    flags.Parse(args)
    // Implementation...
}
```

2. Export the function (capital first letter):

```go
func NewCommand(args []string) { ... }
```

3. Add case in `cmd/adoctl/main.go` switch statement:

```go
case "newcmd":
    prmessagegen.NewCommand(args)
```

4. Add to help text in `main.go`

5. Rebuild and test:

```bash
make build
./dist/adoctl newcmd
```

## Adding a New Feature

1. Identify the appropriate file:
   - `azure_devops_client.go` - API interactions
   - `pr_manager.go` - Business logic
   - `cache.go` - Caching
   - `interactive.go` - User prompts

2. Add your feature with proper error handling

3. Add tests if needed

4. Update documentation in README.md

## Versioning

Version is set via build flags:

- `make build` uses `git describe` for version
- Dev builds show as "dev"

View version:

```bash
make version
# or
adoctl version
```

## Testing

### Unit Tests

Create test files alongside source files:

```
pkg/prmessagegen/myfeature_test.go
```

Example:

```go
package prmessagegen

import "testing"

func TestMyFunction(t *testing.T) {
    result := MyFunction("input")
    if result != "expected" {
        t.Errorf("Expected 'expected', got '%s'", result)
    }
}
```

### Integration Testing

Test with real API using `go run`:

```bash
# Set any required environment variables
export AZURE_DEVOPS_TOKEN="your-token"

# Test specific commands
make dev ARGS="repos"
make dev ARGS="report --creator self"
```

## Debugging

### Verbose Output

```bash
# Run with Go trace
go run -trace=trace.out ./cmd/adoctl repos

# Analyze trace
go tool trace trace.out
```

### Debug Builds

Build without optimization:

```bash
go build -gcflags="all=-N -l" -o adoctl-debug ./cmd/adoctl
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Ensure `make fmt`, `make vet`, and `make test` pass
6. Update README.md if needed
7. Submit a pull request

## Release Process

1. Update version in git tags:

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

2. Build release binaries:

```bash
make build-release
```

3. Binaries are in `dist/` with format:

```
adoctl-{version}_{os}_{arch}
```

4. Attach to GitHub release

## Build Cache

The application maintains a local SQLite database cache for build information from Azure DevOps. This cache is located in the user's cache directory (typically `~/.cache/adoctl/cache.db` on Linux/macOS or `%LOCALAPPDATA%\adoctl\cache.db` on Windows).

### Database Schema

#### Builds Table

The `builds` table stores build information with the following schema:

| Column       | Type     | Description                                                          |
| ------------ | -------- | -------------------------------------------------------------------- |
| `build_id`   | INTEGER  | Primary key, unique build ID from Azure DevOps                       |
| `branch`     | TEXT     | Source branch name (e.g., `main`, `feature/xyz`)                     |
| `repository` | TEXT     | Repository/pipeline name                                             |
| `commit`     | TEXT     | Commit SHA or build number                                           |
| `start_time` | DATETIME | Build start time                                                     |
| `end_time`   | DATETIME | Build end time (NULL if build is still running)                      |
| `status`     | TEXT     | Build status (e.g., `inProgress`, `completed`, `failed`, `canceled`) |
| `full_json`  | TEXT     | Complete JSON response from Azure DevOps API                         |
| `updated_at` | DATETIME | Last time this record was updated                                    |

**Indexes:**

- `idx_builds_status` - For filtering by build status
- `idx_builds_branch` - For filtering by branch
- `idx_builds_repository` - For filtering by repository
- `idx_builds_commit` - For filtering by commit
- `idx_builds_start_time` - For time-based queries

#### Sync Metadata Table

The `sync_metadata` table tracks the last sync time for various data types:

| Column  | Type     | Description                                             |
| ------- | -------- | ------------------------------------------------------- |
| `key`   | TEXT     | Primary key, sync type identifier (e.g., `builds_sync`) |
| `value` | DATETIME | Timestamp of the last successful sync                   |

### Build Sync Strategy

The application uses an efficient sync strategy to minimize API calls:

1. **First Sync**: Fetches all builds and stores them in the cache
2. **Subsequent Syncs**:
   - Fetches only builds that started after the last sync time
3. **Cache Management**:
   - Incomplete builds are updated on each sync until they complete
   - Completed builds are never modified after first insertion

### API Commands

#### Sync Builds

Sync builds from Azure DevOps to local cache:

```bash
# Sync only new and incomplete builds
adoctl sync-builds

# Force sync all builds (ignores cache)
adoctl sync-builds --force
```

#### Search Builds

Search builds in the local cache with various filters:

```bash
# List all builds
adoctl search-builds

# Filter by status
adoctl search-builds --status completed

# Filter by branch
adoctl search-builds --branch main

# Filter by repository
adoctl search-builds --repository my-pipeline

# Filter by commit
adoctl search-builds --commit abc123

# Filter by time range
adoctl search-builds --start-time-from 2024-01-01T00:00:00Z --start-time-to 2024-01-31T23:59:59Z

# Filter by build completion
adoctl search-builds --has-end-time true    # Only completed builds
adoctl search-builds --has-end-time false   # Only running builds

# Combine multiple filters
adoctl search-builds --branch main --status failed --limit 10

# Output as JSON
adoctl search-builds --json

# Save to file
adoctl search-builds --output results.json
```

### Build Data Model

The `Build` struct in `cache.go` represents a build in the cache:

```go
type Build struct {
    BuildID    int           // Build ID
    Branch     string        // Source branch
    Repository string        // Repository name
    Commit     string        // Commit SHA/build number
    StartTime  time.Time     // Build start time
    EndTime    sql.NullTime  // Build end time (null if incomplete)
    Status     string        // Build status
    FullJSON   string        // Complete API response
    UpdatedAt  time.Time     // Last update timestamp
}
```

### Cache Methods

Key cache methods for managing builds:

- `SaveBuild(build Build)` - Insert or update a build record
- `GetBuildByID(buildID int)` - Retrieve a specific build
- `SearchBuilds(filters map[string]any)` - Search builds with filters
- `GetLastSyncTime(key string)` - Get the last sync timestamp
- `SetLastSyncTime(key string, time Time)` - Update the last sync timestamp
- `GetAllBuilds()` - Retrieve all builds
- `GetBuildsByRepository(repository string)` - Get builds for a specific repository
- `GetBuildsByCommit(commit string)` - Get builds for a specific commit
