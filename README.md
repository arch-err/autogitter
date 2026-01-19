# autogitter

Git repository synchronization tool.

## Installation

```bash
go install github.com/arch-err/autogitter/cmd/ag@latest
```

Or download from [releases](https://github.com/arch-err/autogitter/releases).

## Quick Start

1. **Generate and edit your config:**

```bash
ag config
```

This opens your config file in `$EDITOR`. Configure your sources:

```yaml
sources:
  - name: "GitHub"
    source: github.com/your-username
    strategy: manual
    local_path: "~/Git/github"
    repos:
      - your-username/repo1
      - your-username/repo2
```

2. **Sync your repositories:**

```bash
ag sync
```

That's it! Your repos will be cloned to the specified `local_path`.

## Usage

```bash
# Edit config
ag config

# Validate config
ag config --validate

# Sync repositories
ag sync

# Sync with 8 parallel workers
ag sync -j 8

# Prune repos not in config
ag sync --prune

# Add orphaned repos to config
ag sync --add
```

## Documentation

See [full documentation](https://arch-err.github.io/autogitter).

## License

MIT
