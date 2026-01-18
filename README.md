# autogitter

Git repository synchronization tool.

## Installation

```bash
go install github.com/arch-err/autogitter/cmd/ag@latest
```

Or download from [releases](https://github.com/arch-err/autogitter/releases).

## Quick Start

Create `~/.config/autogitter/config.yaml`:

```yaml
sources:
  - name: "Github"
    source: github.com/arch-err
    strategy: manual
    repos:
      - arch-err/autogitter
    local_path: "~/Git/github"
```

Run sync:

```bash
ag sync
```

## Usage

```bash
# Interactive sync
ag sync

# Prune orphaned repos
ag sync --prune

# Add orphaned repos to config
ag sync --add

# Custom config file
ag sync -c /path/to/config.yaml
```

## Documentation

See [full documentation](https://arch-err.github.io/autogitter).

## License

MIT
