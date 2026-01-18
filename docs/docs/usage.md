# Usage

## Commands

### sync

Synchronize repositories according to config.

```bash
ag sync [flags]
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--prune` | `-p` | Delete repos not in config (confirms first) |
| `--add` | `-a` | Add orphaned repos to config |
| `--force` | | Skip confirmation prompts |

**Examples:**

```bash
# Interactive sync (default)
ag sync

# Prune orphaned repos
ag sync --prune

# Add orphaned repos to config
ag sync --add

# Prune without confirmation
ag sync --prune --force
```

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to config file |
| `--debug` | | Enable debug logging |
| `--version` | `-v` | Show version |

## Diff Display

When running `ag sync`, you'll see a colored diff:

- **Green (+)** - Repos that will be cloned
- **Gray** - Existing repos (unchanged)
- **Red (-)** - Orphaned repos (not in config)

## Interactive Mode

By default, all commands are interactive. When orphaned repos are found, you'll be prompted to:

1. **Prune** - Delete the orphaned repos
2. **Add** - Add them to your config
3. **Skip** - Do nothing

Use flags (`-p`, `-a`, `--force`) to skip interactive prompts.
