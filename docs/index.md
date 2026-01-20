<p align="center">
  <img src="assets/logo.jpg" alt="Autogitter Logo" width="150">
</p>

# Autogitter

**Autogitter** (`ag`) is a Git repository synchronization tool that helps you manage multiple repositories across different providers.

## Features

- **Multi-Provider Support** - GitHub, Gitea, and Bitbucket (Cloud + Server)
- **Flexible Sync Strategies** - Manual lists, fetch all from API, or regex pattern matching
- **Parallel Operations** - Clone and pull with configurable worker pools
- **Remote Configs** - Load configuration from HTTP/HTTPS URLs or SSH paths
- **Dry Run Mode** - Preview changes before applying them
- **Interactive CLI** - Beautiful diffs, prompts, and progress indicators
- **SSH Options** - Custom ports and private keys per source

## Quick Start

```bash
# 1. Install
go install github.com/arch-err/autogitter/cmd/ag@latest

# 2. Set up authentication (for all/regex strategies)
ag connect

# 3. Create and edit your config
ag config

# 4. Sync your repositories
ag sync

# 5. Keep repos updated
ag pull
```

## Example Config

```yaml
sources:
  # Sync all repos from a GitHub user
  - name: "GitHub"
    source: github.com/your-username
    strategy: all
    local_path: "~/Git/github"

  # Sync specific repos manually
  - name: "Work"
    source: gitea.company.com/myteam
    strategy: manual
    local_path: "~/Git/work"
    repos:
      - myteam/project-alpha
      - myteam/project-beta

  # Sync repos matching a pattern
  - name: "APIs"
    source: github.com/myorg
    strategy: regex
    local_path: "~/Git/apis"
    regex_strategy:
      pattern: "^myorg/api-.*"

  # Bitbucket Server with custom SSH port
  - name: "Bitbucket"
    source: bitbucket.company.com/~username
    strategy: all
    type: bitbucket
    local_path: "~/Git/bitbucket"
    ssh_options:
      port: 7999
      private_key: "~/.ssh/work_key"
```

## Documentation

- [Installation](installation.md) - How to install autogitter
- [Configuration](configuration.md) - Config file format and options
- [Usage](usage.md) - Commands and flags reference

## Links

- [GitHub Repository](https://github.com/arch-err/autogitter)
- [Releases](https://github.com/arch-err/autogitter/releases)
- [Issues](https://github.com/arch-err/autogitter/issues)
