# Usage

## Commands

### sync

Synchronize repositories according to config. Clones new repos and detects orphaned ones.

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
| `--dry-run` | `-n` | Show what would happen without making changes |

**Examples:**

```bash
# Interactive sync (default)
ag sync

# Preview changes without making them
ag sync --dry-run

# Clone with 8 parallel workers
ag sync -j 8

# Prune orphaned repos
ag sync --prune

# Add orphaned repos to config
ag sync --add

# Prune without confirmation
ag sync --prune --force

# Use a remote config
ag sync -c https://example.com/config.yaml
```

### pull

Pull updates for all local repositories.

```bash
ag pull [flags]
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--force` | | Skip confirmation prompts |
| `--jobs` | `-j` | Number of parallel pull workers (default: 4) |

**Examples:**

```bash
# Pull all repos (interactive)
ag pull

# Pull with 8 parallel workers
ag pull -j 8

# Pull without confirmation
ag pull --force
```

### connect

Configure API authentication for GitHub, Gitea, Bitbucket, or other providers.

```bash
ag connect [flags]
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--type` | `-t` | Connector type (github\|gitea\|bitbucket) |
| `--host` | `-H` | Git server host (e.g., gitea.company.com) |
| `--token` | `-T` | API token (skips interactive prompt) |
| `--list` | `-l` | List configured connections |

**Examples:**

```bash
# Interactive setup (recommended)
ag connect

# List configured connections
ag connect --list

# Non-interactive GitHub setup
ag connect --type github --token ghp_xxxx

# Non-interactive Gitea setup
ag connect --type gitea --host gitea.company.com --token xxxx

# Non-interactive Bitbucket setup
ag connect --type bitbucket --token xxxx

# Bitbucket Server
ag connect --type bitbucket --host bitbucket.company.com --token xxxx
```

Tokens are stored in `$XDG_DATA_HOME/autogitter/credentials.env` (typically `~/.local/share/autogitter/credentials.env`).

### config

Edit or validate the configuration file.

```bash
ag config [flags]
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--validate` | `-v` | Validate config without editing |
| `--generate` | `-g` | Output default config template to stdout |

**Examples:**

```bash
# Edit config in $EDITOR (creates default if missing)
ag config

# Generate config template to stdout
ag config --generate

# Pipe to file
ag config --generate > my-config.yaml

# Validate local config file
ag config --validate

# Validate a specific config file
ag config -v -c /path/to/config.yaml

# Validate a remote config
ag config -v -c https://example.com/config.yaml
```

- Opens config in `$EDITOR` (falls back to `vim`, `nano`, or `vi`)
- Creates a default template if config doesn't exist
- Validates config after editing; prompts to re-edit if invalid

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to config file (local, HTTP, or SSH) |
| `--debug` | | Enable debug logging |
| `--version` | | Show version |
| `--help` | `-h` | Show help |

## Remote Config Support

The `-c` flag accepts local paths, HTTP/HTTPS URLs, or SSH paths:

```bash
# Local file
ag sync -c /path/to/config.yaml

# HTTP/HTTPS
ag sync -c https://example.com/config.yaml
ag sync -c https://raw.githubusercontent.com/user/repo/main/config.yaml

# SSH (both formats supported)
ag sync -c user@host:/path/to/config.yaml
ag sync -c ssh://user@host/path/to/config.yaml
```

## Diff Display

When running `ag sync`, you'll see a colored diff showing what will change:

- **Green (+)** - Repos that will be cloned
- **Gray** - Existing repos (unchanged)
- **Red (-)** - Orphaned repos (not in config)

## Interactive Mode

By default, commands are interactive. When orphaned repos are found during sync, you'll be prompted to:

1. **Prune** - Delete the orphaned repos
2. **Add** - Add them to your config
3. **Skip** - Do nothing

Use flags (`--prune`, `--add`, `--force`) to skip interactive prompts for scripting.

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | Error (config invalid, connection failed, etc.) |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GITHUB_TOKEN` | GitHub API token |
| `GITEA_TOKEN` | Gitea API token |
| `BITBUCKET_TOKEN` | Bitbucket API token |
| `EDITOR` | Preferred editor for `ag config` |

## Scripting Examples

```bash
# Sync all sources, prune orphans, no prompts
ag sync --prune --force

# Validate config in CI
ag config --validate || exit 1

# Pull all repos in cron job
ag pull --force -j 8

# Generate and customize config
ag config --generate > ~/.config/autogitter/config.yaml
vim ~/.config/autogitter/config.yaml

# Use config from dotfiles repo
ag sync -c ~/dotfiles/autogitter.yaml
```
