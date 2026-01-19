# Configuration

The configuration file is stored at `$XDG_CONFIG_HOME/autogitter/config.yaml` (typically `~/.config/autogitter/config.yaml`).

## Config Format

```yaml
sources:
  # Manual strategy - explicitly list repos
  - name: "Github (Personal)"
    source: github.com/username
    strategy: manual
    local_path: "$HOME/Git/github"
    repos:
      - username/repo1
      - username/repo2

  # All strategy - sync all repos from user/org
  - name: "Github (All)"
    source: github.com/username
    strategy: all
    local_path: "$HOME/Git/github-all"

  # Gitea with explicit type override
  - name: "Work Gitea"
    source: gitea.company.com/myuser
    strategy: all
    type: gitea  # explicit type (auto-detected if omitted)
    local_path: "/work/git"
    private_key: "~/.ssh/work_ed25519"
```

## Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Display name for the source |
| `source` | Yes | Git host and user/org (e.g., `github.com/username`) |
| `strategy` | Yes | Sync strategy: `manual`, `all`, or `file` |
| `type` | No | Provider type: `github`, `gitea` (auto-detected from host if omitted) |
| `local_path` | Yes | Where to clone repos (supports `$HOME`, `~`) |
| `repos` | For manual | List of repos to sync |
| `private_key` | No | Path to SSH key for this source |
| `branch` | No | Branch to clone (uses remote default if not set) |

## Strategies

### Manual

Explicitly list repositories to sync:

```yaml
strategy: manual
repos:
  - user/repo1
  - user/repo2
```

### All

Sync all repositories from a user/organization. Requires API authentication - run `ag connect` first.

```yaml
strategy: all
```

This will fetch all non-archived repositories from the specified user or organization.

### File (Coming Soon)

Sync repositories containing a specific file:

```yaml
strategy: file
file_strategy:
  filename: "ag"  # or your custom filename
```

## Authentication

The `all` strategy requires API tokens to list repositories. Set up authentication with:

```bash
ag connect
```

Tokens are stored in `$XDG_DATA_HOME/autogitter/credentials.env` and loaded automatically during sync.

**Environment Variables:**

| Provider | Environment Variable |
|----------|---------------------|
| GitHub | `GITHUB_TOKEN` |
| Gitea | `GITEA_TOKEN` |

You can also export these variables directly:

```bash
export GITHUB_TOKEN=ghp_xxxx
ag sync
```

## Environment Variables

Paths support environment variable expansion:

- `$HOME` or `~` - User's home directory
- `$XDG_CONFIG_HOME` - XDG config directory
- Any other environment variable

## Custom Config Path

Use the `-c` or `--config` flag to specify a custom config file:

```bash
ag sync -c /path/to/config.yaml
```
