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
# Install
go install github.com/arch-err/autogitter/cmd/ag@latest

# Create config
mkdir -p ~/.config/autogitter
cat > ~/.config/autogitter/config.yaml << 'EOF'
sources:
  - name: "Github"
    source: github.com/your-username
    strategy: manual
    repos:
      - your-username/repo1
      - your-username/repo2
    local_path: "~/Git/github"
EOF

# Sync
ag sync
```

## Documentation

- [Installation](installation.md) - How to install autogitter
- [Configuration](configuration.md) - Config file format and options
- [Usage](usage.md) - Commands and flags
