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
| `--jobs` | `-j` | Number of parallel clone workers (default: 4) |

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

# Clone with 8 parallel workers
ag sync -j 8
```

### config

Edit or validate the configuration file.

```bash
ag config [flags]
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--validate` | `-v` | Validate config without editing |
| `--generate` | `-g` | Generate default config file (fails if exists) |

**Examples:**

```bash
# Edit config in $EDITOR (creates default if missing)
ag config

# Generate config without opening editor
ag config --generate

# Validate config file
ag config --validate

# Validate a specific config file
ag config -v -c /path/to/config.yaml
```

- Opens config in `$EDITOR` (falls back to `vim`, `nano`, or `vi`)
- Creates a default template if config doesn't exist
- Validates config after editing; prompts to re-edit if invalid

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
