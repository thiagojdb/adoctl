# Contributing to adoctl

Thank you for your interest in contributing to adoctl! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Pull Request Process](#pull-request-process)
- [Coding Standards](#coding-standards)
- [Testing Requirements](#testing-requirements)
- [Commit Message Guidelines](#commit-message-guidelines)

## Code of Conduct

- Be respectful and inclusive
- Provide constructive feedback
- Focus on the best solution for the project
- Welcome newcomers and help them get started

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/adoctl.git
   cd adoctl
   ```
3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/ORIGINAL_OWNER/adoctl.git
   ```
4. **Install dependencies**:
   ```bash
   go mod download
   ```
5. **Install development tools**:
   ```bash
   # Install golangci-lint
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

## Development Workflow

1. **Create a feature branch** from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**:
   - Write clean, readable code
   - Follow existing code style and patterns
   - Add tests for new functionality
   - Update documentation as needed

3. **Test your changes**:
   ```bash
   # Run tests
   make test

   # Run linter
   make lint

   # Check formatting
   make fmt
   ```

4. **Commit your changes**:
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```

5. **Keep your branch up to date**:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

6. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```

7. **Open a Pull Request** on GitHub

## Pull Request Process

1. **Before submitting**:
   - Ensure all tests pass locally
   - Run the linter and fix any issues
   - Update documentation if needed
   - Add tests for new features or bug fixes

2. **PR title and description**:
   - Use a clear, descriptive title
   - Reference any related issues
   - Describe what changes were made and why
   - Include screenshots for UI changes

3. **PR requirements**:
   - All CI checks must pass
   - Code coverage should not decrease
   - At least one approval from a maintainer
   - No unresolved review comments

4. **After PR is merged**:
   - Delete your feature branch
   - Update your local main branch:
     ```bash
     git checkout main
     git pull upstream main
     ```

## Coding Standards

### Go Style

- Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` for formatting (run `make fmt`)
- Run `go vet` to catch common mistakes (run `make vet`)
- Use meaningful variable and function names
- Keep functions small and focused (max ~50 lines)
- Add comments for exported functions and complex logic

### Project Patterns

- **Error Handling**: Use the `pkg/errors` package for typed errors with exit codes
- **Logging**: Use the `pkg/logger` package (zerolog) for structured logging
  - Use `logger.Debug()` for detailed diagnostics
  - Use `logger.Info()` for user-facing operations
  - Use `logger.Warn()` for non-fatal errors
  - Use `logger.Error()` for fatal errors
- **Configuration**: All config goes through `pkg/config`
- **Never suppress errors silently**: Always log warnings or return errors

### Code Organization

```
adoctl/
├── cmd/                    # CLI commands (Cobra)
│   ├── root.go            # Root command and flags
│   ├── pr_*.go            # PR-related commands
│   ├── build_*.go         # Build-related commands
│   └── ...
├── pkg/
│   ├── config/            # Configuration management
│   ├── logger/            # Structured logging
│   ├── errors/            # Typed errors
│   ├── cache/             # SQLite caching
│   ├── azure/client/      # Azure DevOps API client
│   ├── devops/            # Business logic layer
│   └── utils/             # Utility functions
└── docs/                  # Documentation
```

## Testing Requirements

### Coverage

- Maintain minimum 70% code coverage
- All new features must include tests
- Bug fixes should include regression tests

### Test Types

1. **Unit Tests** (`*_test.go` files):
   ```go
   func TestFunctionName(t *testing.T) {
       // Arrange
       input := "test"

       // Act
       result := FunctionName(input)

       // Assert
       if result != expected {
           t.Errorf("got %v, want %v", result, expected)
       }
   }
   ```

2. **Integration Tests** (with `// +build integration` tag):
   ```go
   // +build integration

   func TestIntegration_Feature(t *testing.T) {
       // Test with real Azure DevOps API
   }
   ```

3. **Table-Driven Tests** (preferred for multiple scenarios):
   ```go
   func TestMultipleScenarios(t *testing.T) {
       tests := []struct {
           name     string
           input    string
           expected string
       }{
           {"scenario 1", "input1", "output1"},
           {"scenario 2", "input2", "output2"},
       }

       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               result := Function(tt.input)
               if result != tt.expected {
                   t.Errorf("got %v, want %v", result, tt.expected)
               }
           })
       }
   }
   ```

### Running Tests

```bash
# Unit tests only
make test

# With race detection
go test -race ./...

# With coverage
make test-cover

# Integration tests (requires AZURE_PAT)
go test -tags=integration ./pkg/integration/...
```

## Commit Message Guidelines

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `refactor`: Code refactoring (no functional changes)
- `perf`: Performance improvements
- `test`: Adding or updating tests
- `chore`: Maintenance tasks (dependencies, build, etc.)
- `ci`: CI/CD changes

### Examples

```
feat(pr): add bulk PR creation command

Add ability to create PRs across multiple repositories with the same
source and target branches. Useful for organization-wide changes.

Closes #123
```

```
fix(cache): prevent silent cache save failures

Previously, cache save errors were silently ignored with continue.
Now they are logged as warnings so users can see when caching fails.

Fixes #456
```

```
refactor(config): extract helper functions

Extracted loadConfigFile, applyEnvironmentOverrides, and validateConfig
to eliminate 47 lines of duplication between Load() and loadFromPath().
```

## Questions?

- Open an issue for bugs or feature requests
- Start a discussion for questions or ideas
- Check existing issues before creating new ones

Thank you for contributing! 🎉
