# Autogitter

**Autogitter** (`ag`) is a Git repository synchronization tool that helps you manage multiple repositories across different sources.

## Features

- Sync repositories from GitHub, Gitea, and other Git providers
- Multiple strategies: manual, all, file-based
- Interactive CLI with beautiful output
- Automatic diff display showing what will change
- Prune orphaned repos or add them to config

## Quick Start

```bash
# 1. Install
go install github.com/arch-err/autogitter/cmd/ag@latest

# 2. Generate and edit your config
ag config

# 3. Sync your repositories
ag sync
```

The `ag config` command will create a default config template and open it in your editor. Configure your sources, save, and you're ready to sync!

## Documentation

- [Installation](installation.md) - How to install autogitter
- [Configuration](configuration.md) - Config file format and options
- [Usage](usage.md) - Commands and flags
