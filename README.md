# adoctl - Azure DevOps CLI Tool

![CI](https://github.com/thiagojdb/adoctl/workflows/CI/badge.svg)

> Stop fighting Azure DevOps web UI. Get work done from your terminal.

**adoctl** is a fast, focused CLI for Azure DevOps that respects your time. Query work items, manage pipelines, and inspect repos without leaving your keyboard.

---

## Quick Start

```bash
# Install
go install github.com/thiagojdb/adoctl@latest

# Configure
export AZURE_PAT="your_token"
adoctl repos

# Get to work
adoctl pr list
adoctl pr create --title "My feature"
```

---

## Features

- **PR Management**: Create, list, merge, approve pull requests
- **Work Items**: Query, link, track with natural filters
- **Pipelines**: Monitor builds and deployments
- **Fast**: Local SQLite cache, parallel API calls
- **Unix-friendly**: Pipe to `jq`, grep, or your scripts

---

## Installation

### Go Install

```bash
go install github.com/thiagojdb/adoctl@latest
```

### Build from source

```bash
git clone https://github.com/thiagojdb/adoctl.git
cd adoctl
make build
```

---

## Usage

```bash
adoctl [command] [flags]

Commands:
  pr          Pull requests (list, create, merge)
  repos       List repositories
  pipeline    Monitor builds
```

---

## Configuration

```bash
# Environment variables
export AZURE_ORG="yourorg"
export AZURE_PROJECT="yourproject"
export AZURE_PAT="your-personal-access-token"

# Or use config file
cp config.example.yaml ~/.config/adoctl/config.yaml
```

---

## Development

```bash
# Build
make build

# Test
make test

# Run locally
make dev ARGS="pr list"
```

---

## License

MIT
